package agent

import (
	"time"

	boshalert "github.com/cloudfoundry/bosh-agent/agent/alert"
	boshas "github.com/cloudfoundry/bosh-agent/agent/applier/applyspec"
	bosherr "github.com/cloudfoundry/bosh-agent/errors"
	boshhandler "github.com/cloudfoundry/bosh-agent/handler"
	boshjobsuper "github.com/cloudfoundry/bosh-agent/jobsupervisor"
	boshlog "github.com/cloudfoundry/bosh-agent/logger"
	boshplatform "github.com/cloudfoundry/bosh-agent/platform"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	boshsyslog "github.com/cloudfoundry/bosh-agent/syslog"
	boshtime "github.com/cloudfoundry/bosh-agent/time"
	boshuuid "github.com/cloudfoundry/bosh-agent/uuid"
)

const (
	agentLogTag = "agent"
)

type Agent struct {
	logger            boshlog.Logger
	mbusHandler       boshhandler.Handler
	platform          boshplatform.Platform
	actionDispatcher  ActionDispatcher
	heartbeatInterval time.Duration
	jobSupervisor     boshjobsuper.JobSupervisor
	specService       boshas.V1Service
	syslogServer      boshsyslog.Server
	settingsService   boshsettings.Service
	uuidGenerator     boshuuid.Generator
	timeService       boshtime.Service
}

func New(
	logger boshlog.Logger,
	mbusHandler boshhandler.Handler,
	platform boshplatform.Platform,
	actionDispatcher ActionDispatcher,
	jobSupervisor boshjobsuper.JobSupervisor,
	specService boshas.V1Service,
	syslogServer boshsyslog.Server,
	heartbeatInterval time.Duration,
	settingsService boshsettings.Service,
	uuidGenerator boshuuid.Generator,
	timeService boshtime.Service,
) Agent {
	return Agent{
		logger:            logger,
		mbusHandler:       mbusHandler,
		platform:          platform,
		actionDispatcher:  actionDispatcher,
		heartbeatInterval: heartbeatInterval,
		jobSupervisor:     jobSupervisor,
		specService:       specService,
		syslogServer:      syslogServer,
		settingsService:   settingsService,
		uuidGenerator:     uuidGenerator,
		timeService:       timeService,
	}
}

func (a Agent) Run() error {
	a.logger.Debug(agentLogTag, "Starting monit")
	err := a.platform.StartMonit()
	if err != nil {
		return bosherr.WrapError(err, "Starting Monit")
	}

	errCh := make(chan error, 1)

	a.actionDispatcher.ResumePreviouslyDispatchedTasks()

	go a.subscribeActionDispatcher(errCh)

	go a.generateHeartbeats(errCh)

	go a.jobSupervisor.MonitorJobFailures(a.handleJobFailure(errCh))

	go a.syslogServer.Start(a.handleSyslogMsg(errCh))

	select {
	case err = <-errCh:
		return err
	}
}

func (a Agent) subscribeActionDispatcher(errCh chan error) {
	defer a.logger.HandlePanic("Agent Message Bus Handler")

	err := a.mbusHandler.Run(a.actionDispatcher.Dispatch)
	if err != nil {
		err = bosherr.WrapError(err, "Message Bus Handler")
	}

	errCh <- err
}

func (a Agent) generateHeartbeats(errCh chan error) {
	a.logger.Debug(agentLogTag, "Generating heartbeat")
	defer a.logger.HandlePanic("Agent Generate Heartbeats")

	// Send initial heartbeat
	a.sendHeartbeat(errCh)

	tickChan := time.Tick(a.heartbeatInterval)

	for {
		select {
		case <-tickChan:
			a.sendHeartbeat(errCh)
		}
	}
}

func (a Agent) sendHeartbeat(errCh chan error) {
	heartbeat, err := a.getHeartbeat()
	if err != nil {
		err = bosherr.WrapError(err, "Building heartbeat")
		errCh <- err
		return
	}

	err = a.mbusHandler.Send(boshhandler.HealthMonitor, boshhandler.Heartbeat, heartbeat)
	if err != nil {
		err = bosherr.WrapError(err, "Sending heartbeat")
		errCh <- err
	}
}

func (a Agent) getHeartbeat() (Heartbeat, error) {
	a.logger.Debug(agentLogTag, "Building heartbeat")
	vitalsService := a.platform.GetVitalsService()

	vitals, err := vitalsService.Get()
	if err != nil {
		return Heartbeat{}, bosherr.WrapError(err, "Getting job vitals")
	}

	spec, err := a.specService.Get()
	if err != nil {
		return Heartbeat{}, bosherr.WrapError(err, "Getting job spec")
	}

	hb := Heartbeat{
		Job:      spec.JobSpec.Name,
		Index:    spec.Index,
		JobState: a.jobSupervisor.Status(),
		Vitals:   vitals,
	}
	return hb, nil
}

func (a Agent) handleJobFailure(errCh chan error) boshjobsuper.JobFailureHandler {
	return func(monitAlert boshalert.MonitAlert) error {
		alertAdapter := boshalert.NewMonitAdapter(monitAlert, a.settingsService, a.timeService)
		if alertAdapter.IsIgnorable() {
			a.logger.Debug(agentLogTag, "Ignored monit event: ", monitAlert.Event)
			return nil
		}

		severity, found := alertAdapter.Severity()
		if !found {
			a.logger.Error(agentLogTag, "Unknown monit event name `%s', using default severity %d", monitAlert.Event, severity)
		}

		alert, err := alertAdapter.Alert()
		if err != nil {
			errCh <- bosherr.WrapError(err, "Adapting monit alert")
		}

		err = a.mbusHandler.Send(boshhandler.HealthMonitor, boshhandler.Alert, alert)
		if err != nil {
			errCh <- bosherr.WrapError(err, "Sending monit alert")
		}

		return nil
	}
}

func (a Agent) handleSyslogMsg(errCh chan error) boshsyslog.CallbackFunc {
	return func(msg boshsyslog.Msg) {
		alertAdapter := boshalert.NewSSHAdapter(
			msg,
			a.settingsService,
			a.uuidGenerator,
			a.timeService,
			a.logger,
		)
		if alertAdapter.IsIgnorable() {
			a.logger.Debug(agentLogTag, "Ignored ssh event: ", msg.Content)
			return
		}

		alert, err := alertAdapter.Alert()
		if err != nil {
			errCh <- bosherr.WrapError(err, "Adapting SSH alert")
		}

		err = a.mbusHandler.Send(boshhandler.HealthMonitor, boshhandler.Alert, alert)
		if err != nil {
			errCh <- bosherr.WrapError(err, "Sending SSH alert")
		}
	}
}

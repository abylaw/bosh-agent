package platform

import (
	"time"

	sigar "github.com/cloudfoundry/gosigar"

	bosherror "github.com/cloudfoundry/bosh-agent/errors"
	boshlog "github.com/cloudfoundry/bosh-agent/logger"
	boshcdrom "github.com/cloudfoundry/bosh-agent/platform/cdrom"
	boshudev "github.com/cloudfoundry/bosh-agent/platform/cdrom/udevdevice"
	boshcmd "github.com/cloudfoundry/bosh-agent/platform/commands"
	boshdisk "github.com/cloudfoundry/bosh-agent/platform/disk"
	boshnet "github.com/cloudfoundry/bosh-agent/platform/net"
	bosharp "github.com/cloudfoundry/bosh-agent/platform/net/arp"
	boship "github.com/cloudfoundry/bosh-agent/platform/net/ip"
	boshstats "github.com/cloudfoundry/bosh-agent/platform/stats"
	boshvitals "github.com/cloudfoundry/bosh-agent/platform/vitals"
	boshdirs "github.com/cloudfoundry/bosh-agent/settings/directories"
	boshsys "github.com/cloudfoundry/bosh-agent/system"
)

const (
	ArpIterations          = 20
	ArpIterationDelay      = 5 * time.Second
	ArpInterfaceCheckDelay = 100 * time.Millisecond
)

const (
	SigarStatsCollectionInterval = 10 * time.Second
)

type provider struct {
	platforms map[string]Platform
}

type ProviderOptions struct {
	Linux LinuxOptions
}

func NewProvider(logger boshlog.Logger, dirProvider boshdirs.Provider, options ProviderOptions) (p provider) {
	runner := boshsys.NewExecCmdRunner(logger)
	fs := boshsys.NewOsFileSystem(logger)

	linuxDiskManager := boshdisk.NewLinuxDiskManager(logger, runner, fs, options.Linux.BindMountPersistentDisk)

	udev := boshudev.NewConcreteUdevDevice(runner, logger)
	linuxCdrom := boshcdrom.NewLinuxCdrom("/dev/sr0", udev, runner)
	linuxCdutil := boshcdrom.NewCdUtil(dirProvider.SettingsDir(), fs, linuxCdrom, logger)

	compressor := boshcmd.NewTarballCompressor(runner, fs)
	copier := boshcmd.NewCpCopier(runner, fs, logger)

	sigarCollector := boshstats.NewSigarStatsCollector(&sigar.ConcreteSigar{})

	// Kick of stats collection as soon as possible
	go sigarCollector.StartCollecting(SigarStatsCollectionInterval, nil)

	vitalsService := boshvitals.NewService(sigarCollector, dirProvider)

	routesSearcher := boshnet.NewCmdRoutesSearcher(runner)
	ipResolver := boship.NewResolver(boship.NetworkInterfaceToAddrsFunc)

	defaultNetworkResolver := boshnet.NewDefaultNetworkResolver(routesSearcher, ipResolver)
	arping := bosharp.NewArping(runner, fs, logger, ArpIterations, ArpIterationDelay, ArpInterfaceCheckDelay)

	centosNetManager := boshnet.NewCentosNetManager(fs, runner, defaultNetworkResolver, ipResolver, arping, logger)
	ubuntuNetManager := boshnet.NewUbuntuNetManager(fs, runner, defaultNetworkResolver, ipResolver, arping, logger)

	centos := NewLinuxPlatform(
		fs,
		runner,
		sigarCollector,
		compressor,
		copier,
		dirProvider,
		vitalsService,
		linuxCdutil,
		linuxDiskManager,
		centosNetManager,
		500*time.Millisecond,
		options.Linux,
		logger,
	)

	ubuntu := NewLinuxPlatform(
		fs,
		runner,
		sigarCollector,
		compressor,
		copier,
		dirProvider,
		vitalsService,
		linuxCdutil,
		linuxDiskManager,
		ubuntuNetManager,
		500*time.Millisecond,
		options.Linux,
		logger,
	)

	p.platforms = map[string]Platform{
		"ubuntu": ubuntu,
		"centos": centos,
		"dummy":  NewDummyPlatform(sigarCollector, fs, runner, dirProvider, logger),
	}
	return
}

func (p provider) Get(name string) (Platform, error) {
	plat, found := p.platforms[name]
	if !found {
		return nil, bosherror.New("Platform %s could not be found", name)
	}
	return plat, nil
}

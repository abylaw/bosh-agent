package fakes

import (
	boshdpresolv "github.com/cloudfoundry/bosh-agent/infrastructure/devicepathresolver"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
)

type FakeInfrastructure struct {
	Settings                boshsettings.Settings
	SetupSSHUsername        string
	SetupNetworkingNetworks boshsettings.Networks

	GetEphemeralDiskPathDevicePath string
	GetEphemeralDiskPathFound      bool
	GetEphemeralDiskPathRealPath   string

	DevicePathResolver boshdpresolv.DevicePathResolver
}

func NewFakeInfrastructure() (infrastructure *FakeInfrastructure) {
	infrastructure = &FakeInfrastructure{}
	infrastructure.Settings = boshsettings.Settings{}
	return
}

func (i *FakeInfrastructure) GetDevicePathResolver() (devicePathResolver boshdpresolv.DevicePathResolver) {
	return i.DevicePathResolver
}

func (i *FakeInfrastructure) SetupSSH(username string) (err error) {
	i.SetupSSHUsername = username
	return
}

func (i *FakeInfrastructure) GetSettings() (settings boshsettings.Settings, err error) {
	settings = i.Settings
	return
}

func (i *FakeInfrastructure) SetupNetworking(networks boshsettings.Networks) (err error) {
	i.SetupNetworkingNetworks = networks
	return
}

func (i *FakeInfrastructure) GetEphemeralDiskPath(devicePath string) string {
	i.GetEphemeralDiskPathDevicePath = devicePath
	return i.GetEphemeralDiskPathRealPath
}

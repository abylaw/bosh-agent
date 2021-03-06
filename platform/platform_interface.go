package platform

import (
	boshdpresolv "github.com/cloudfoundry/bosh-agent/infrastructure/devicepathresolver"
	boshcmd "github.com/cloudfoundry/bosh-agent/platform/commands"
	boshvitals "github.com/cloudfoundry/bosh-agent/platform/vitals"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	boshdir "github.com/cloudfoundry/bosh-agent/settings/directories"
	boshsys "github.com/cloudfoundry/bosh-agent/system"
)

type Platform interface {
	GetFs() boshsys.FileSystem
	GetRunner() boshsys.CmdRunner
	GetCompressor() boshcmd.Compressor
	GetCopier() boshcmd.Copier
	GetDirProvider() boshdir.Provider
	GetVitalsService() boshvitals.Service

	GetDevicePathResolver() (devicePathResolver boshdpresolv.DevicePathResolver)
	SetDevicePathResolver(devicePathResolver boshdpresolv.DevicePathResolver) error

	// User management
	CreateUser(username, password, basePath string) (err error)
	AddUserToGroups(username string, groups []string) (err error)
	DeleteEphemeralUsersMatching(regex string) (err error)

	// Bootstrap functionality
	SetupSSH(publicKey, username string) (err error)
	SetUserPassword(user, encryptedPwd string) (err error)
	SetupHostname(hostname string) (err error)
	SetupDhcp(networks boshsettings.Networks) (err error)
	SetupManualNetworking(networks boshsettings.Networks) (err error)
	SetupLogrotate(groupName, basePath, size string) (err error)
	SetTimeWithNtpServers(servers []string) (err error)
	SetupEphemeralDiskWithPath(devicePath string) (err error)
	SetupDataDir() (err error)
	SetupTmpDir() (err error)
	SetupMonitUser() (err error)
	StartMonit() (err error)
	SetupRuntimeConfiguration() (err error)

	// Disk management
	MountPersistentDisk(devicePath, mountPoint string) error
	UnmountPersistentDisk(devicePath string) (didUnmount bool, err error)
	MigratePersistentDisk(fromMountPoint, toMountPoint string) (err error)
	NormalizeDiskPath(devicePath string) string
	IsMountPoint(path string) (result bool, err error)
	IsPersistentDiskMounted(path string) (result bool, err error)

	GetFileContentsFromCDROM(filePath string) (contents []byte, err error)
	GetFilesContentsFromDisk(diskPath string, fileNames []string) (contents [][]byte, err error)

	// Network misc
	PrepareForNetworkingChange() error
	GetDefaultNetwork() (boshsettings.Network, error)

	// Additional monit management
	GetMonitCredentials() (username, password string, err error)
}

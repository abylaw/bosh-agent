package devicepathresolver

type dummyDevicePathResolver struct{}

func NewDummyDevicePathResolver() dummyDevicePathResolver {
	return dummyDevicePathResolver{}
}

func (resolver dummyDevicePathResolver) GetRealDevicePath(devicePath string) (string, bool, error) {
	return devicePath, false, nil
}

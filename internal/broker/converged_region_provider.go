package broker

type ConvergedCloudRegionProvider interface {
	GetRegions() []string
}

type PathBasedConvergedCloudRegionsProvider struct {
	// placeholder
}

func (c *PathBasedConvergedCloudRegionsProvider) GetRegions() []string {
	return []string{"eu-de-1"}
}

type OneForAllConvergedCloudRegionsProvider struct {
}

func (c *OneForAllConvergedCloudRegionsProvider) GetRegions() []string {
	return []string{"eu-de-1"}
}

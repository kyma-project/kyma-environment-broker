package broker

import (
	"fmt"
)

//go:generate mockery --name=RegionReader --output=automock --outpkg=automock --case=underscore
type RegionReader interface {
	Read(filename string) (map[string][]string, error)
}

type ConvergedCloudRegionProvider interface {
	GetRegions(string) []string
}

type DefaultConvergedCloudRegionsProvider struct {
	// placeholder
	regionConfiguration map[string][]string
}

func NewPathBasedConvergedCloudRegionsProvider(regionConfigurationPath string, reader RegionReader) (*DefaultConvergedCloudRegionsProvider, error) {
	regionConfiguration, err := reader.Read(regionConfigurationPath)
	if err != nil {
		return nil, fmt.Errorf("while unmarshalling a file with sap-converged-cloud region mappings: %w", err)
	}

	return &DefaultConvergedCloudRegionsProvider{
		regionConfiguration: regionConfiguration,
	}, nil
}

func (c *DefaultConvergedCloudRegionsProvider) GetRegions(mappedRegion string) []string {
	item, found := c.regionConfiguration[mappedRegion]

	if !found {
		return []string{}
	}

	return item
}

type OneForAllConvergedCloudRegionsProvider struct {
}

func (c *OneForAllConvergedCloudRegionsProvider) GetRegions(mappedRegion string) []string {
	return []string{"eu-de-1"}
}

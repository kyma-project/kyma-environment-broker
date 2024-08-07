package provider

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/stretchr/testify/assert"
)

func TestZonesForSapConvergedCloudZones(t *testing.T) {
	convergedCloudRegionProvider := broker.OneForAllConvergedCloudRegionsProvider{}
	regions := convergedCloudRegionProvider.GetRegions("")
	for _, region := range regions {
		_, exists := sapConvergedCloudZones[region]
		assert.True(t, exists)
	}
	_, exists := sapConvergedCloudZones[DefaultSapConvergedCloudRegion]
	assert.True(t, exists)
}

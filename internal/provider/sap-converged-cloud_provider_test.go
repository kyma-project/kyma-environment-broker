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

func TestMultipleZonesForSapConvergedCloudRegion(t *testing.T) {
	t.Run("for valid zonesCount", func(t *testing.T) {
		// given
		region := "eu-de-1"

		// when
		generatedZones := ZonesForSapConvergedCloud(region, 3)

		// then
		for _, zone := range generatedZones {
			regionFromZone := zone[:len(zone)-1]
			assert.Equal(t, region, regionFromZone)
		}
		assert.Equal(t, 3, len(generatedZones))
		// check if all zones are unique
		assert.Condition(t, func() (success bool) {
			var zones []string
			for _, zone := range generatedZones {
				for _, z := range zones {
					if zone == z {
						return false
					}
				}
				zones = append(zones, zone)
			}
			return true
		})
	})

	t.Run("for zonesCount exceeding maximum zones for region", func(t *testing.T) {
		// given
		region := "eu-de-1"
		zonesCountExceedingMaximum := 20
		maximumZonesForRegion := len(sapConvergedCloudZones[region])
		// "eu-de-1" region has maximum 3 zones, user request 20

		// when
		generatedZones := ZonesForSapConvergedCloud(region, zonesCountExceedingMaximum)

		// then
		for _, zone := range generatedZones {
			regionFromZone := zone[:len(zone)-1]
			assert.Equal(t, region, regionFromZone)
		}
		assert.Equal(t, maximumZonesForRegion, len(generatedZones))
	})
}

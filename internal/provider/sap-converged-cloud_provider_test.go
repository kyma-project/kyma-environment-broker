package provider

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/stretchr/testify/assert"
)

func TestZonesForSapConvergedCloudZones(t *testing.T) {
	regions := broker.SapConvergedCloudRegions()
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

func TestSapConvergedCloudInput_SingleZone_ApplyParameters(t *testing.T) {
	// given
	svc := SapConvergedCloudInput{}

	// when
	t.Run("use default region and default zones count", func(t *testing.T) {
		// given
		input := svc.Defaults()

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{})

		//then
		assert.Equal(t, DefaultSapConvergedCloudRegion, input.GardenerConfig.Region)
		assert.Len(t, input.GardenerConfig.ProviderSpecificConfig.OpenStackConfig.Zones, 1)

		for _, zone := range input.GardenerConfig.ProviderSpecificConfig.OpenStackConfig.Zones {
			regionFromZone := zone[:len(zone)-1]
			assert.Equal(t, DefaultSapConvergedCloudRegion, regionFromZone)
		}
	})

	// when
	t.Run("use region input parameter", func(t *testing.T) {
		// given
		input := svc.Defaults()
		inputRegion := "eu-de-1"

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{
				Region: ptr.String(inputRegion),
			},
		})

		//then
		assert.Len(t, input.GardenerConfig.ProviderSpecificConfig.OpenStackConfig.Zones, 1)

		for _, zone := range input.GardenerConfig.ProviderSpecificConfig.OpenStackConfig.Zones {
			regionFromZone := zone[:len(zone)-1]
			assert.Equal(t, inputRegion, regionFromZone)
		}
	})

	// when
	t.Run("use zones list input parameter", func(t *testing.T) {
		// given
		input := svc.Defaults()
		zones := []string{"eu-de-1a", "eu-de-1b"}

		// when
		svc.ApplyParameters(input, internal.ProvisioningParameters{
			Parameters: internal.ProvisioningParametersDTO{
				Zones: zones,
			},
		})

		//then
		assert.Len(t, input.GardenerConfig.ProviderSpecificConfig.OpenStackConfig.Zones, len(zones))

		for i, zone := range input.GardenerConfig.ProviderSpecificConfig.OpenStackConfig.Zones {
			assert.Equal(t, zones[i], zone)
		}
	})
}

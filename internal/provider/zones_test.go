package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMultipleZonesForSapConvergedCloudRegion(t *testing.T) {
	for tname, tcase := range map[string]struct {
		region     string
		zonesCount int
	}{
		"for valid zonesCount in eu-de-1": {
			region:     "eu-de-1",
			zonesCount: 3,
		},
		"for valid zonesCount in ap-au-1": {
			region:     "ap-au-1",
			zonesCount: 2,
		},
		"for valid zonesCount in na-us-1": {
			region:     "na-us-1",
			zonesCount: 3,
		},
		"for valid zonesCount in eu-de-2": {
			region:     "eu-de-2",
			zonesCount: 2,
		},
		"for valid zonesCount in na-us-2": {
			region:     "na-us-2",
			zonesCount: 2,
		},
		"for valid zonesCount in ap-jp-1": {
			region:     "ap-jp-1",
			zonesCount: 1,
		},
		"for valid zonesCount in ap-ae-1": {
			region:     "ap-ae-1",
			zonesCount: 2,
		},
	} {
		t.Run(tname, func(t *testing.T) {
			// when
			generatedZones := ZonesForSapConvergedCloud(tcase.region, tcase.zonesCount)

			// then
			for _, zone := range generatedZones {
				regionFromZone := zone[:len(zone)-1]
				assert.Equal(t, tcase.region, regionFromZone)
			}
			assert.Equal(t, tcase.zonesCount, len(generatedZones))
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
	}

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

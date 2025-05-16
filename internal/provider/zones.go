package provider

import (
	"fmt"
	"math/rand"

	"github.com/kyma-project/kyma-environment-broker/common/runtime"
)

func GenerateAzureZones(zonesCount int) []string {
	zones := []string{"1", "2", "3"}
	if zonesCount > 3 {
		zonesCount = 3
	}

	rand.Shuffle(len(zones), func(i, j int) { zones[i], zones[j] = zones[j], zones[i] })
	return zones[:zonesCount]
}

// sapConvergedCloudZones defines a possible suffixes for given OpenStack regions
// The table is tested in a unit test to check if all necessary regions are covered
var sapConvergedCloudZones = map[string]string{
	"eu-de-1": "abd",
	"ap-au-1": "ab",
	"na-us-1": "abd",
	"eu-de-2": "ab",
	"na-us-2": "ab",
	"ap-jp-1": "a",
	"ap-ae-1": "ab",
}

func CountZonesForSapConvergedCloud(region string) int {
	zones, found := sapConvergedCloudZones[region]
	if !found {
		return 0
	}

	return len(zones)
}

func ZonesForSapConvergedCloud(region string, availableZones []string) []string {
	var generatedZones []string
	for _, zone := range availableZones {
		generatedZones = append(generatedZones, fmt.Sprintf("%s%s", region, zone))
	}
	return generatedZones
}

type zonesProviderMock struct {
	zones []string
}

func (z *zonesProviderMock) RandomZones(cp runtime.CloudProvider, region string, zonesCount int) []string {
	if zonesCount < len(z.zones) {
		return z.zones[:zonesCount]
	}
	return z.zones
}

func FakeZonesProvider(zones []string) *zonesProviderMock {
	return &zonesProviderMock{
		zones: zones,
	}
}

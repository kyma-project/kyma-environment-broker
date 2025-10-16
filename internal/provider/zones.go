package provider

import (
	"fmt"
	"math/rand"
	"unicode"

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
		generatedZones = append(generatedZones, FullZoneName(OpenstackProviderType, region, zone))
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

func FullZoneName(providerType string, region string, zone string) string {
	switch providerType {
	case GCPProviderType:
		return fmt.Sprintf("%s-%s", region, zone)
	case AzureProviderType:
		return fmt.Sprintf("%s", zone)
	case OpenstackProviderType:
		return fmt.Sprintf("%s%s", region, zone)
	case AWSProviderType:
		return fmt.Sprintf("%s%s", region, zone)
	case AlicloudProviderType:
		if isDigit(rune(region[len(region)-1])) {
			return fmt.Sprintf("%s%s", region, zone)
		} else {
			return fmt.Sprintf("%s-%s", region, zone)
		}
	}
	return zone
}

func isDigit(r rune) bool {
	return unicode.IsDigit(r)
}

package provider

import (
	"fmt"
	"math/rand"
	"strings"
)

func GenerateAzureZones(zonesCount int) []string {
	zones := []string{"1", "2", "3"}
	if zonesCount > 3 {
		zonesCount = 3
	}

	rand.Shuffle(len(zones), func(i, j int) { zones[i], zones[j] = zones[j], zones[i] })
	return zones[:zonesCount]
}

func MultipleZonesForAWSRegion(region string, zonesCount int) []string {
	zones, found := awsZones[region]
	if !found {
		zones = "a"
		zonesCount = 1
	}

	availableZones := strings.Split(zones, "")
	rand.Shuffle(len(availableZones), func(i, j int) { availableZones[i], availableZones[j] = availableZones[j], availableZones[i] })
	if zonesCount > len(availableZones) {
		// get maximum number of zones for region
		zonesCount = len(availableZones)
	}

	availableZones = availableZones[:zonesCount]

	var generatedZones []string
	for _, zone := range availableZones {
		generatedZones = append(generatedZones, fmt.Sprintf("%s%s", region, zone))
	}
	return generatedZones
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

func ZonesForSapConvergedCloud(region string, zonesCount int) []string {
	zones, found := sapConvergedCloudZones[region]
	if !found {
		zones = "a"
		zonesCount = 1
	}

	availableZones := strings.Split(zones, "")
	rand.Shuffle(len(availableZones), func(i, j int) { availableZones[i], availableZones[j] = availableZones[j], availableZones[i] })
	if zonesCount > len(availableZones) {
		// get maximum number of zones for region
		zonesCount = len(availableZones)
	}

	availableZones = availableZones[:zonesCount]

	var generatedZones []string
	for _, zone := range availableZones {
		generatedZones = append(generatedZones, fmt.Sprintf("%s%s", region, zone))
	}
	return generatedZones
}

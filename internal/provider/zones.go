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

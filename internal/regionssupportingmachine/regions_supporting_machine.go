package regionssupportingmachine

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/utils"
)

type RegionsSupportingMachine map[string]map[string][]string

func ReadRegionsSupportingMachineFromFile(filename string, zoneMapping bool) (RegionsSupportingMachine, error) {
	var regionsSupportingMachineWithZones RegionsSupportingMachine
	if zoneMapping {
		err := utils.UnmarshalYamlFile(filename, &regionsSupportingMachineWithZones)
		if err != nil {
			return RegionsSupportingMachine{}, fmt.Errorf("while unmarshalling a file with regions supporting machine extended with zone mapping: %w", err)
		}
	} else {
		regionsSupportingMachine := make(map[string][]string)
		err := utils.UnmarshalYamlFile(filename, &regionsSupportingMachine)
		if err != nil {
			return RegionsSupportingMachine{}, fmt.Errorf("while unmarshalling a file with regions supporting machine: %w", err)
		}
		regionsSupportingMachineWithZones = convert(regionsSupportingMachine)
	}
	return regionsSupportingMachineWithZones, nil
}

func (r RegionsSupportingMachine) IsSupported(region string, machineType string) bool {
	for machineFamily, regions := range r {
		if strings.HasPrefix(machineType, machineFamily) {
			if _, exists := regions[region]; exists {
				return true
			}
			return false
		}
	}

	return true
}

func (r RegionsSupportingMachine) SupportedRegions(machineType string) []string {
	for machineFamily, regionsMap := range r {
		if strings.HasPrefix(machineType, machineFamily) {
			regions := make([]string, 0, len(regionsMap))
			for region := range regionsMap {
				regions = append(regions, region)
			}
			sort.Strings(regions)
			return regions
		}
	}
	return []string{}
}

func (r RegionsSupportingMachine) AvailableZones(machineType, region, planID string) ([]string, error) {
	for machineFamily, regionsMap := range r {
		if strings.HasPrefix(machineType, machineFamily) {
			zones := regionsMap[region]
			if len(zones) == 0 {
				return []string{}, nil
			}
			rand.Shuffle(len(zones), func(i, j int) { zones[i], zones[j] = zones[j], zones[i] })
			if len(zones) > 3 {
				zones = zones[:3]
			}

			switch planID {
			case broker.AWSPlanID, broker.BuildRuntimeAWSPlanID, broker.PreviewPlanID, broker.SapConvergedCloudPlanID:
				var formattedZones []string
				for _, zone := range zones {
					formattedZones = append(formattedZones, fmt.Sprintf("%s%s", region, zone))
				}
				return formattedZones, nil

			case broker.AzurePlanID, broker.BuildRuntimeAzurePlanID:
				return zones, nil

			case broker.AzureLitePlanID:
				return zones[:1], nil

			case broker.GCPPlanID, broker.BuildRuntimeGCPPlanID:
				var formattedZones []string
				for _, zone := range zones {
					formattedZones = append(formattedZones, fmt.Sprintf("%s-%s", region, zone))
				}
				return formattedZones, nil

			default:
				return []string{}, fmt.Errorf("plan %s not supported", planID)
			}
		}
	}

	return []string{}, nil
}

func convert(input map[string][]string) RegionsSupportingMachine {
	output := make(RegionsSupportingMachine)

	for machineType, regions := range input {
		output[machineType] = make(map[string][]string)
		for _, region := range regions {
			output[machineType][region] = nil
		}
	}

	return output
}

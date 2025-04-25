package regionssupportingmachine

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"

	"github.com/kyma-project/kyma-environment-broker/internal/utils"
)

const (
	GCPPlanID               = "ca6e5357-707f-4565-bbbd-b3ab732597c6"
	AWSPlanID               = "361c511f-f939-4621-b228-d0fb79a1fe15"
	AzurePlanID             = "4deee563-e5ec-4731-b9b1-53b42d855f0c"
	AzureLitePlanID         = "8cb22518-aa26-44c5-91a0-e669ec9bf443"
	SapConvergedCloudPlanID = "03b812ac-c991-4528-b5bd-08b303523a63"
	PreviewPlanID           = "5cb3d976-b85c-42ea-a636-79cadda109a9"
	BuildRuntimeAWSPlanID   = "6aae0ff3-89f7-4f12-86de-51466145422e"
	BuildRuntimeGCPPlanID   = "a310cd6b-6452-45a0-935d-d24ab53f9eba"
	BuildRuntimeAzurePlanID = "499244b4-1bef-48c9-be68-495269899f8e"
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
			case AWSPlanID, BuildRuntimeAWSPlanID, PreviewPlanID, SapConvergedCloudPlanID:
				var formattedZones []string
				for _, zone := range zones {
					formattedZones = append(formattedZones, fmt.Sprintf("%s%s", region, zone))
				}
				return formattedZones, nil

			case AzurePlanID, BuildRuntimeAzurePlanID:
				return zones, nil

			case AzureLitePlanID:
				return zones[:1], nil

			case GCPPlanID, BuildRuntimeGCPPlanID:
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

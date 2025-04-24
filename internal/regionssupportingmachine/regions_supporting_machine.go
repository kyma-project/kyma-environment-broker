package regionssupportingmachine

import (
	"fmt"
	"sort"
	"strings"

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

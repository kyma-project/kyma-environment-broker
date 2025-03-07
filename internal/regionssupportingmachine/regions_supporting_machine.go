package regionssupportingmachine

import (
	"fmt"
	"strings"

	"github.com/kyma-project/kyma-environment-broker/internal/utils"
)

func ReadRegionsSupportingMachineFromFile(filename string) (map[string][]string, error) {
	regionsSupportingMachine := make(map[string][]string)
	err := utils.UnmarshalYamlFile(filename, &regionsSupportingMachine)
	if err != nil {
		return map[string][]string{}, fmt.Errorf("while unmarshalling a file with regions supporting machine: %w", err)
	}
	return regionsSupportingMachine, nil
}

func IsSupported(regionsSupportingMachine map[string][]string, region string, machineType string) bool {
	for machineFamily, regions := range regionsSupportingMachine {
		if strings.HasPrefix(machineType, machineFamily) {
			for _, r := range regions {
				if r == region {
					return true
				}
			}
			return false
		}
	}

	return true
}

func SupportedRegions(regionsSupportingMachine map[string][]string, machineType string) []string {
	for machineFamily, regions := range regionsSupportingMachine {
		if strings.HasPrefix(machineType, machineFamily) {
			return regions
		}
	}
	return []string{}
}

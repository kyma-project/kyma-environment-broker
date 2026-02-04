package gardener

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func ReadDNSProvidersValuesFromYAML(yamlFilePath string) (DNSProvidersData, error) {
	var values DNSProvidersData
	yamlFile, err := os.ReadFile(yamlFilePath)
	if err != nil {
		return DNSProvidersData{}, fmt.Errorf("while reading YAML file with DNS default values: %w", err)
	}

	err = yaml.Unmarshal(yamlFile, &values)
	if err != nil {
		return DNSProvidersData{}, fmt.Errorf("while unmarshalling YAML file with DNS default values: %w", err)

	}

	return values, nil
}

package btpregionsmigration

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

func ReadFromFile(filename string) (map[string]string, error) {
	btpRegionsMigrationSapConvergedCloud, err := os.ReadFile(filename)
	if err != nil {
		return map[string]string{}, fmt.Errorf("while reading %s file with deprecated BTP regions for SAP Converged Cloud: %w", filename, err)
	}
	var data map[string]string
	err = yaml.Unmarshal(btpRegionsMigrationSapConvergedCloud, &data)
	if err != nil {
		return map[string]string{}, fmt.Errorf("while unmarshalling a file with deprecated BTP regions for SAP Converged Cloud: %w", err)
	}
	return data, nil
}

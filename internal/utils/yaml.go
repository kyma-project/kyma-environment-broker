package utils

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func UnmarshalYamlFile(filename string, out interface{}) error {
	var fileBytes, err = os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("while reading a %s file : %w", filename, err)
	}

	err = yaml.Unmarshal(fileBytes, out)
	if err != nil {
		return fmt.Errorf("while unmarshaling yaml data: %w", err)
	}

	return nil
}

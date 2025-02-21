package model

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type RulesConfig struct {
	Rules []string `yaml:"rule"`
}

func (c *RulesConfig) Load(file string) (*RulesConfig, error) {

	yamlFile, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML file: %s", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML file: %s", err)
	}

	return c, nil
}

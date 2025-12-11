package config

import (
	"gopkg.in/yaml.v3"
)

type ConfigMapConverter struct{}

func NewConfigMapConverter() *ConfigMapConverter {
	return &ConfigMapConverter{}
}

func (c *ConfigMapConverter) Convert(from string, to any) error {
	if err := yaml.Unmarshal([]byte(from), to); err != nil {
		return err
	}
	return nil
}

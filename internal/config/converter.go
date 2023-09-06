package config

import (
	"github.com/kyma-project/kyma-environment-broker/internal"
	"gopkg.in/yaml.v2"
)

type ConfigMapConverter struct{}

func NewConfigMapConverter() *ConfigMapConverter {
	return &ConfigMapConverter{}
}

func (c *ConfigMapConverter) ConvertToStruct(cfgString string) (internal.ConfigForPlan, error) {
	var cfg internal.ConfigForPlan
	if err := yaml.Unmarshal([]byte(cfgString), &cfg); err != nil {
		return internal.ConfigForPlan{}, err
	}
	return cfg, nil
}

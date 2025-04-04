package config

import (
	"github.com/kyma-project/kyma-environment-broker/internal"
)

type (
	ConfigurationProvider interface {
		ProvideForGivenPlan(planName string) (*internal.ConfigForPlan, error)
	}

	ConfigReader interface {
		Read(planName string) (string, error)
	}

	ConfigValidator interface {
		Validate(cfgString string) error
	}

	ConfigConverter interface {
		ConvertToStruct(cfgString string) (internal.ConfigForPlan, error)
	}
)

type ConfigProvider struct {
	Reader    ConfigReader
	Validator ConfigValidator
	Converter ConfigConverter
}

func NewConfigProvider(reader ConfigReader, validator ConfigValidator, converter ConfigConverter) *ConfigProvider {
	return &ConfigProvider{Reader: reader, Validator: validator, Converter: converter}
}

func (p *ConfigProvider) ProvideForGivenPlan(planName string) (*internal.ConfigForPlan, error) {
	cfgString, err := p.Reader.Read(planName)
	if err != nil {
		return nil, err
	}

	if err = p.Validator.Validate(cfgString); err != nil {
		return nil, err
	}

	cfg, err := p.Converter.ConvertToStruct(cfgString)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

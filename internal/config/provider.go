package config

import (
	"github.com/kyma-project/kyma-environment-broker/internal"
)

type (
	Provider interface {
		ProvideForGivenPlan(planName string) (*internal.ConfigForPlan, error)
	}

	Reader interface {
		Read(objectName, configKey string) (string, error)
	}

	Validator interface {
		Validate(requiredFields, cfgString string) error
	}

	Converter interface {
		Convert(from string, to any) error
	}
)

type provider struct {
	Reader    Reader
	Validator Validator
	Converter Converter
}

func NewConfigProvider(reader Reader, validator Validator, converter Converter) Provider {
	return &provider{Reader: reader, Validator: validator, Converter: converter}
}

func (p *provider) ProvideForGivenPlan(planName string) (*internal.ConfigForPlan, error) {
	cfgString, err := p.Reader.Read("", planName)
	if err != nil {
		return nil, err
	}

	if err = p.Validator.Validate(runtimeConfigurationRequiredFields, cfgString); err != nil {
		return nil, err
	}

	var cfg internal.ConfigForPlan
	err = p.Converter.Convert(cfgString, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

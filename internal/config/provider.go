package config

type (
	Provider interface {
		Provide(cfgSrcName, cfgKeyName, reqCfgKeys string, cfgDestObj any) error
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

func (p *provider) Provide(cfgSrcName, cfgKeyName, reqCfgKeys string, cfgDestObj any) error {
	cfgString, err := p.Reader.Read(cfgSrcName, cfgKeyName)
	if err != nil {
		return err
	}

	if err = p.Validator.Validate(reqCfgKeys, cfgString); err != nil {
		return err
	}

	err = p.Converter.Convert(cfgString, cfgDestObj)
	if err != nil {
		return err
	}

	return nil
}

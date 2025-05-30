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

	ConfigMapConfigProvider interface {
		Provide(cfgKeyName string, cfgDestObj any) error
	}
)

type configMapConfigProvider struct {
	Provider
	configMapName string
	reqCfgKeys    string
}

func NewConfigMapConfigProvider(provider Provider, configMapName string, reqCfgKeys string) ConfigMapConfigProvider {
	return &configMapConfigProvider{
		Provider:      provider,
		configMapName: configMapName,
		reqCfgKeys:    reqCfgKeys,
	}
}

func (p *configMapConfigProvider) Provide(cfgKeyName string, cfgDestObj any) error {
	return p.Provider.Provide(p.configMapName, cfgKeyName, p.reqCfgKeys, cfgDestObj)
}

type provider struct {
	reader    Reader
	validator Validator
	converter Converter
}

func NewConfigProvider(reader Reader, validator Validator, converter Converter) Provider {
	return &provider{reader: reader, validator: validator, converter: converter}
}

func (p *provider) Provide(cfgSrcName, cfgKeyName, reqCfgKeys string, cfgDestObj any) error {
	cfgString, err := p.reader.Read(cfgSrcName, cfgKeyName)
	if err != nil {
		return err
	}

	if err = p.validator.Validate(reqCfgKeys, cfgString); err != nil {
		return err
	}

	err = p.converter.Convert(cfgString, cfgDestObj)
	if err != nil {
		return err
	}

	return nil
}

package testutil

import "github.com/kyma-project/kyma-environment-broker/internal/config"

// FakeProvider implements config.Provider for testing
type FakeProvider struct{}

func (f *FakeProvider) Provide(cfgSrcName, cfgKeyName, reqCfgKeys string, cfgDestObj any) error {
	if cfgKeyName == "default" {
		mockConfig := map[string]interface{}{
			"kyma-template": `apiVersion: operator.kyma-project.io/v1beta2
kind: Kyma
metadata:
  name: default
spec:
  channel: regular
  modules:
    - name: btp-operator
      channel: regular
    - name: keda
      channel: fast
`,
		}

		if configMap, ok := cfgDestObj.(*map[string]interface{}); ok {
			*configMap = mockConfig
		}
		return nil
	}

	if configMap, ok := cfgDestObj.(*map[string]interface{}); ok {
		*configMap = make(map[string]interface{})
	}
	return nil
}

// NewFakeConfigProvider returns a new instance of the fake config provider
func NewFakeConfigProvider() config.Provider {
	return &FakeProvider{}
}

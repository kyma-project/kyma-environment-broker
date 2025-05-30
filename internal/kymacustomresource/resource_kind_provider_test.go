package kymacustomresource

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const kymaTemplate = `apiVersion: operator.kyma-project.io/v1beta2
kind: Kyma
metadata:
  name: kyma-template
  labels:
    "label1": "value1"
`

func TestResourceKindProvider(t *testing.T) {
	// given
	fakeCfgProvider := fakeConfigProvider{}
	resourceKindProvider := NewResourceKindProvider(fakeCfgProvider)

	expectedGroup := "operator.kyma-project.io"
	expectedVersion := "v1beta2"
	expectedKind := "Kyma"
	expectedResource := "kymas"

	// when
	gvk, err := resourceKindProvider.DefaultGvk()
	require.NoError(t, err)

	// then
	assert.Equal(t, expectedGroup, gvk.Group)
	assert.Equal(t, expectedVersion, gvk.Version)
	assert.Equal(t, expectedKind, gvk.Kind)

	// when
	gvr, err := resourceKindProvider.DefaultGvr()
	require.NoError(t, err)

	// then
	assert.Equal(t, expectedGroup, gvr.Group)
	assert.Equal(t, expectedVersion, gvr.Version)
	assert.Equal(t, expectedResource, gvr.Resource)

}

type fakeConfigProvider struct{}

func (f fakeConfigProvider) Provide(cfgKeyName string, cfgDestObj any) error {
	cfg, _ := cfgDestObj.(*internal.ConfigForPlan)
	cfg.KymaTemplate = kymaTemplate
	cfgDestObj = cfg
	return nil
}

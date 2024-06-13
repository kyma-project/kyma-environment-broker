package broker

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/broker/automock"
	"github.com/stretchr/testify/assert"
)

func TestOneForAllConvergedCloudRegionsProvider_GetDefaultRegions(t *testing.T) {
	// given
	c := &OneForAllConvergedCloudRegionsProvider{}

	// when
	result := c.GetRegions("")

	// then
	assert.Equal(t, []string{"eu-de-1"}, result)
}

func TestPathBasedConvergedCloudRegionsProvider_FactoryMethod(t *testing.T) {
	// given
	configLocation := "path-to-config"
	regions := map[string][]string{
		"key": {"value"},
	}

	mockReader := automock.NewRegionReader(t)
	mockReader.On("Read", configLocation).Return(regions, nil)

	// when
	provider, err := NewPathBasedConvergedCloudRegionsProvider(configLocation, mockReader)

	// then
	assert.NoError(t, err)
	assert.Equal(t, regions, provider.regionConfiguration)
}

func TestPathBasedConvergedCloudRegionsProvider_GetRegions(t *testing.T) {

	t.Run("should return existing mapping in one item configuration", func(t *testing.T) {
		// given
		regions := map[string][]string{
			"key": {"value"},
		}

		provider := &DefaultConvergedCloudRegionsProvider{
			regionConfiguration: regions,
		}

		// when
		result := provider.GetRegions("key")

		// then
		assert.NotNil(t, result)
		assert.Len(t, result, 1)
		assert.Equal(t, result[0], "value")
	})

	t.Run("should return existing mappings in multi item configuration", func(t *testing.T) {
		// given
		regions := map[string][]string{
			"key1": {"value1"},
			"key2": {"value2"},
			"key3": {"value3", "value4"},
		}

		provider := &DefaultConvergedCloudRegionsProvider{
			regionConfiguration: regions,
		}

		// when
		result := provider.GetRegions("key2")

		// then
		assert.NotNil(t, result)
		assert.Len(t, result, 1)
		assert.Equal(t, result[0], "value2")

		// when
		result = provider.GetRegions("key3")

		// then
		assert.NotNil(t, result)
		assert.Len(t, result, 2)
		assert.Equal(t, result[0], "value3")
		assert.Equal(t, result[1], "value4")

	})

	t.Run("should return empty region list for non-existing mapping in one item configuration", func(t *testing.T) {
		// given
		regions := map[string][]string{
			"key": {"value"},
		}

		provider := &DefaultConvergedCloudRegionsProvider{
			regionConfiguration: regions,
		}

		// when
		result := provider.GetRegions("key-does-not-exist")

		// then
		assert.NotNil(t, result)
		assert.Len(t, result, 0)
	})
}

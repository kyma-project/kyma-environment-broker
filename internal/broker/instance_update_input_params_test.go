package broker

import (
	"encoding/json"
	"testing"

	"github.com/pivotal-cf/brokerapi/v12/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdatingParameters_Gvisor(t *testing.T) {
	endpoint := &UpdateEndpoint{}

	t.Run("should unmarshal gvisor enabled: true", func(t *testing.T) {
		// given
		rawParams := json.RawMessage(`{"gvisor": {"enabled": true}}`)
		details := domain.UpdateDetails{RawParameters: rawParams}

		// when
		params, err := endpoint.unmarshalParams(details, fixLogger())

		// then
		require.NoError(t, err)
		require.NotNil(t, params.Gvisor)
		assert.True(t, params.Gvisor.Enabled)
	})

	t.Run("should return nil gvisor when key is absent", func(t *testing.T) {
		// given
		rawParams := json.RawMessage(`{}`)
		details := domain.UpdateDetails{RawParameters: rawParams}

		// when
		params, err := endpoint.unmarshalParams(details, fixLogger())

		// then
		require.NoError(t, err)
		assert.Nil(t, params.Gvisor)
	})

	t.Run("should unmarshal gvisor enabled: false", func(t *testing.T) {
		// given
		rawParams := json.RawMessage(`{"gvisor": {"enabled": false}}`)
		details := domain.UpdateDetails{RawParameters: rawParams}

		// when
		params, err := endpoint.unmarshalParams(details, fixLogger())

		// then
		require.NoError(t, err)
		require.NotNil(t, params.Gvisor)
		assert.False(t, params.Gvisor.Enabled)
	})
}

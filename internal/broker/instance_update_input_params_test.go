package broker

import (
	"encoding/json"
	"testing"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/whitelist"
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

func TestPersistenceOnUpdate_Gvisor(t *testing.T) {
	endpoint := &UpdateEndpoint{}
	gvisor := &pkg.GvisorDTO{Enabled: true}

	// given
	instance := &internal.Instance{}
	params := &internal.UpdatingParametersDTO{Gvisor: gvisor}
	operation := &internal.Operation{}
	details := domain.UpdateDetails{}

	// when
	_, err := endpoint.updateInstanceAndOperationParameters(instance, params, operation, details, internal.ERSContext{}, fixLogger())

	// then
	require.NoError(t, err)
	assert.Equal(t, gvisor, instance.Parameters.Parameters.Gvisor)
}

func TestGvisorWhitelist_Update(t *testing.T) {
	const allowedGA = "allowed-global-account-id"
	const otherGA = "other-global-account-id"
	gvisorEnabled := &pkg.GvisorDTO{Enabled: true}
	gvisorDisabled := &pkg.GvisorDTO{Enabled: false}

	t.Run("should allow when gvisor is disabled and whitelist is empty (default)", func(t *testing.T) {
		// given
		endpoint := &UpdateEndpoint{gvisorWhitelist: whitelist.Set{}}

		// when
		err := endpoint.validateGvisorWhitelist(gvisorDisabled, otherGA)

		// then
		require.NoError(t, err)
	})

	t.Run("should allow when gvisor is disabled and global account is not in whitelist", func(t *testing.T) {
		// given
		endpoint := &UpdateEndpoint{gvisorWhitelist: whitelist.Set{allowedGA: {}}}

		// when
		err := endpoint.validateGvisorWhitelist(gvisorDisabled, otherGA)

		// then
		require.NoError(t, err)
	})

	t.Run("should reject when gvisor is enabled and global account is not in whitelist", func(t *testing.T) {
		// given
		endpoint := &UpdateEndpoint{gvisorWhitelist: whitelist.Set{allowedGA: {}}}

		// when
		err := endpoint.validateGvisorWhitelist(gvisorEnabled, otherGA)

		// then
		require.Error(t, err)
	})

	t.Run("should allow when gvisor is enabled and global account is in whitelist", func(t *testing.T) {
		// given
		endpoint := &UpdateEndpoint{gvisorWhitelist: whitelist.Set{allowedGA: {}}}

		// when
		err := endpoint.validateGvisorWhitelist(gvisorEnabled, allowedGA)

		// then
		require.NoError(t, err)
	})

	t.Run("should reject when gvisor enabled and whitelist is empty", func(t *testing.T) {
		// given
		endpoint := &UpdateEndpoint{gvisorWhitelist: whitelist.Set{}}

		// when
		err := endpoint.validateGvisorWhitelist(gvisorEnabled, allowedGA)

		// then
		require.Error(t, err)
	})
}

package broker

import (
	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/whitelist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_gvisorToBool(t *testing.T) {
	type args struct {
		gvisor *runtime.GvisorDTO
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "returns false for nil",
			args: args{
				gvisor: nil,
			},
			want: false,
		},
		{
			name: "returns false for disabled gvisor",
			args: args{
				gvisor: &runtime.GvisorDTO{Enabled: false},
			},
			want: false,
		},
		{
			name: "returns true for enabled gvisor",
			args: args{
				gvisor: &runtime.GvisorDTO{Enabled: true},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, gvisorToBool(tt.args.gvisor), "gvisorToBool(%v)", tt.args.gvisor)
		})
	}
}

func Test_validateGvisorWhitelist(t *testing.T) {
	// given
	const allowedGA = "allowed-global-account-id"
	const otherGA = "other-global-account-id"
	gvisorEnabled := true
	gvisorDisabled := false

	t.Run("should allow when gvisor is disabled and whitelist is empty (default)", func(t *testing.T) {
		// when
		err := validateGvisorWhitelist(gvisorDisabled, otherGA, whitelist.Set{})

		// then
		require.NoError(t, err)
	})

	t.Run("should allow when gvisor is disabled and global account is not in whitelist", func(t *testing.T) {
		// when
		err := validateGvisorWhitelist(gvisorDisabled, otherGA, whitelist.Set{allowedGA: {}})

		// then
		require.NoError(t, err)
	})

	t.Run("should reject when gvisor is enabled and global account is not in whitelist", func(t *testing.T) {
		// when
		err := validateGvisorWhitelist(gvisorEnabled, otherGA, whitelist.Set{allowedGA: {}})

		// then
		require.Error(t, err)
	})

	t.Run("should allow when gvisor is enabled and global account is in whitelist", func(t *testing.T) {
		// when
		err := validateGvisorWhitelist(gvisorEnabled, allowedGA, whitelist.Set{allowedGA: {}})

		// then
		require.NoError(t, err)
	})

	t.Run("should reject when gvisor is enabled and whitelist is empty", func(t *testing.T) {
		// when
		err := validateGvisorWhitelist(gvisorEnabled, allowedGA, whitelist.Set{})

		// then
		require.Error(t, err)
	})
}

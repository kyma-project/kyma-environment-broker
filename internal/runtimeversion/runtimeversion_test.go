package runtimeversion

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/stretchr/testify/require"
)

func Test_RuntimeVersionConfigurator_ForProvisioning_FromParameters(t *testing.T) {
	t.Run("should return version from Defaults when version not provided", func(t *testing.T) {
		// given
		runtimeVer := "1.1.1"
		operation := internal.Operation{
			ProvisioningParameters: internal.ProvisioningParameters{},
		}
		rvc := NewRuntimeVersionConfigurator(runtimeVer, nil)

		// when
		ver, err := rvc.ForProvisioning(operation)

		// then
		require.NoError(t, err)
		require.Equal(t, runtimeVer, ver.Version)
		require.Equal(t, 1, ver.MajorVersion)
		require.Equal(t, internal.Defaults, ver.Origin)
	})
}

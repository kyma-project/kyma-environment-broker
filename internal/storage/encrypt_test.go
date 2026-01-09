package storage

import (
	"encoding/json"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/rand"
)

type testDto struct {
	Data string `json:"data"`
}

func TestInvalidKey(t *testing.T) {
	secretKey := "1"

	e := NewEncrypter(secretKey)
	dto := testDto{
		Data: secretKey,
	}

	j, err := json.Marshal(&dto)
	require.NoError(t, err)

	t.Run("invalid key for GCM write mode", func(t *testing.T) {
		_, err = e.Encrypt(j)
		require.Error(t, err)
	})
}

func TestDecryptUsingGCMMode(t *testing.T) {
	secretKey := rand.String(32)
	e := NewEncrypter(secretKey)

	data := []byte("test data for GCM decryption")
	encrypted, err := e.encryptGCM(data)
	require.NoError(t, err)

	decrypted, err := e.DecryptUsingMode(encrypted)
	require.NoError(t, err)
	assert.Equal(t, data, decrypted)
}

func TestDecryptSMCredentialsUsingGCMMode(t *testing.T) {
	secretKey := rand.String(32)
	e := NewEncrypter(secretKey)

	params := &internal.ProvisioningParameters{
		ErsContext: internal.ERSContext{
			SMOperatorCredentials: &internal.ServiceManagerOperatorCredentials{
				ClientID:     "gcm-client-id",
				ClientSecret: "gcm-client-secret",
				URL:          "https://example.com",
			},
		},
	}

	err := e.EncryptSMCredentials(params)
	require.NoError(t, err)

	encryptedClientID := params.ErsContext.SMOperatorCredentials.ClientID
	encryptedClientSecret := params.ErsContext.SMOperatorCredentials.ClientSecret
	assert.NotEqual(t, "gcm-client-id", encryptedClientID)
	assert.NotEqual(t, "gcm-client-secret", encryptedClientSecret)

	err = e.DecryptSMCredentialsUsingMode(params)
	require.NoError(t, err)
	assert.Equal(t, "gcm-client-id", params.ErsContext.SMOperatorCredentials.ClientID)
	assert.Equal(t, "gcm-client-secret", params.ErsContext.SMOperatorCredentials.ClientSecret)
}

func TestDecryptSMCredentialsUsingModeWithNilCredentials(t *testing.T) {
	secretKey := rand.String(32)
	e := NewEncrypter(secretKey)

	params := &internal.ProvisioningParameters{
		ErsContext: internal.ERSContext{
			SMOperatorCredentials: nil,
		},
	}

	err := e.DecryptSMCredentialsUsingMode(params)
	require.NoError(t, err)
	assert.Nil(t, params.ErsContext.SMOperatorCredentials)
}

func TestDecryptSMCredentialsUsingModeWithEmptyCredentials(t *testing.T) {
	secretKey := rand.String(32)
	e := NewEncrypter(secretKey)

	params := &internal.ProvisioningParameters{
		ErsContext: internal.ERSContext{
			SMOperatorCredentials: &internal.ServiceManagerOperatorCredentials{
				ClientID:     "",
				ClientSecret: "",
			},
		},
	}

	err := e.DecryptSMCredentialsUsingMode(params)
	require.NoError(t, err)
	assert.Equal(t, "", params.ErsContext.SMOperatorCredentials.ClientID)
	assert.Equal(t, "", params.ErsContext.SMOperatorCredentials.ClientSecret)
}

func TestDecryptKubeconfigUsingGCMMode(t *testing.T) {
	secretKey := rand.String(32)
	e := NewEncrypter(secretKey)

	params := &internal.ProvisioningParameters{

		Parameters: runtime.ProvisioningParametersDTO{
			Kubeconfig: "kubeconfig-gcm-content",
		},
	}

	err := e.EncryptKubeconfig(params)
	require.NoError(t, err)

	encryptedKubeconfig := params.Parameters.Kubeconfig
	assert.NotEqual(t, "kubeconfig-gcm-content", encryptedKubeconfig)

	err = e.DecryptKubeconfigUsingMode(params)
	require.NoError(t, err)
	assert.Equal(t, "kubeconfig-gcm-content", params.Parameters.Kubeconfig)
}

func TestDecryptKubeconfigUsingModeWithEmptyKubeconfig(t *testing.T) {
	secretKey := rand.String(32)
	e := NewEncrypter(secretKey)

	params := &internal.ProvisioningParameters{
		Parameters: runtime.ProvisioningParametersDTO{
			Kubeconfig: "",
		},
	}

	err := e.DecryptKubeconfigUsingMode(params)
	require.NoError(t, err)
	assert.Equal(t, "", params.Parameters.Kubeconfig)
}

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

func TestNewEncrypterInCFBOnlyMode(t *testing.T) {
	secretKey := rand.String(32)
	encrypter := NewEncrypter(secretKey, false)

	t.Run("encrypt json", func(t *testing.T) {
		dto := testDto{
			Data: secretKey,
		}

		j, err := json.Marshal(&dto)
		require.NoError(t, err)

		cipherText, err := encrypter.Encrypt(j)
		require.NoError(t, err)
		assert.NotEqual(t, j, cipherText)

		cipherText, err = encrypter.decryptCFB(cipherText)
		require.NoError(t, err)
		assert.Equal(t, j, cipherText)

		_, err = encrypter.decryptGCM(cipherText)
		require.Error(t, err)

		err = json.Unmarshal(cipherText, &dto)
		require.NoError(t, err)
	})

	t.Run("encrypt string", func(t *testing.T) {
		dto := []byte("test")

		cipherText, err := encrypter.Encrypt(dto)
		require.NoError(t, err)
		assert.NotEqual(t, dto, cipherText)

		cipherText, err = encrypter.decryptCFB(cipherText)
		require.NoError(t, err)
		assert.Equal(t, dto, cipherText)

		_, err = encrypter.decryptGCM(cipherText)
		require.Error(t, err)
	})
}

func TestNewEncrypterInGCMWriteMode(t *testing.T) {
	secretKey := rand.String(32)

	e := NewEncrypter(secretKey, true)

	t.Run("encrypt json", func(t *testing.T) {

		dto := testDto{
			Data: secretKey,
		}

		j, err := json.Marshal(&dto)
		require.NoError(t, err)

		cipherText, err := e.Encrypt(j)
		require.NoError(t, err)
		assert.NotEqual(t, j, cipherText)

		cipherText, err = e.decryptGCM(cipherText)
		require.NoError(t, err)
		assert.Equal(t, j, cipherText)

		_, err = e.decryptCFB(cipherText)
		require.Error(t, err)

		err = json.Unmarshal(cipherText, &dto)
		require.NoError(t, err)
	})

	t.Run("encrypt string", func(t *testing.T) {
		dto := []byte("test")

		cipherText, err := e.Encrypt(dto)
		require.NoError(t, err)
		assert.NotEqual(t, dto, cipherText)

		cipherText, err = e.decryptGCM(cipherText)
		require.NoError(t, err)
		assert.Equal(t, dto, cipherText)

		_, err = e.decryptCFB(cipherText)
		require.Error(t, err)
	})

}

func TestInvalidKey(t *testing.T) {
	secretKey := "1"

	e := NewEncrypter(secretKey, false)
	dto := testDto{
		Data: secretKey,
	}

	j, err := json.Marshal(&dto)
	require.NoError(t, err)

	t.Run("invalid key for CFB mode", func(t *testing.T) {

		_, err = e.Encrypt(j)
		require.Error(t, err)
	})

	t.Run("invalid key for GCM write mode", func(t *testing.T) {
		e.SetWriteGCMMode(true)
		_, err = e.Encrypt(j)
		require.Error(t, err)
	})
}

func TestDecryptUsingCFBMode(t *testing.T) {
	secretKey := rand.String(32)
	e := NewEncrypter(secretKey, false)

	data := []byte("test data for CFB decryption")
	encrypted, err := e.encryptCFB(data)
	require.NoError(t, err)

	decrypted, err := e.DecryptUsingMode(encrypted, EncryptionModeCFB)
	require.NoError(t, err)
	assert.Equal(t, data, decrypted)
}

func TestDecryptUsingGCMMode(t *testing.T) {
	secretKey := rand.String(32)
	e := NewEncrypter(secretKey, true)

	data := []byte("test data for GCM decryption")
	encrypted, err := e.encryptGCM(data)
	require.NoError(t, err)

	decrypted, err := e.DecryptUsingMode(encrypted, EncryptionModeGCM)
	require.NoError(t, err)
	assert.Equal(t, data, decrypted)
}

func TestDecryptUsingModeWithDefaultFallbackToCFB(t *testing.T) {
	secretKey := rand.String(32)
	e := NewEncrypter(secretKey, false)

	data := []byte("test data with unknown mode")
	encrypted, err := e.encryptCFB(data)
	require.NoError(t, err)

	decrypted, err := e.DecryptUsingMode(encrypted, "unknown-mode")
	require.NoError(t, err)
	assert.Equal(t, data, decrypted)
}

func TestDecryptSMCredentialsUsingCFBMode(t *testing.T) {
	secretKey := rand.String(32)
	e := NewEncrypter(secretKey, false)

	params := &internal.ProvisioningParameters{
		ErsContext: internal.ERSContext{
			SMOperatorCredentials: &internal.ServiceManagerOperatorCredentials{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				URL:          "https://example.com",
			},
		},
	}

	err := e.EncryptSMCredentials(params)
	require.NoError(t, err)

	encryptedClientID := params.ErsContext.SMOperatorCredentials.ClientID
	encryptedClientSecret := params.ErsContext.SMOperatorCredentials.ClientSecret
	assert.NotEqual(t, "test-client-id", encryptedClientID)
	assert.NotEqual(t, "test-client-secret", encryptedClientSecret)

	err = e.DecryptSMCredentialsUsingMode(params, EncryptionModeCFB)
	require.NoError(t, err)
	assert.Equal(t, "test-client-id", params.ErsContext.SMOperatorCredentials.ClientID)
	assert.Equal(t, "test-client-secret", params.ErsContext.SMOperatorCredentials.ClientSecret)
	assert.Equal(t, "https://example.com", params.ErsContext.SMOperatorCredentials.URL)
}

func TestDecryptSMCredentialsUsingGCMMode(t *testing.T) {
	secretKey := rand.String(32)
	e := NewEncrypter(secretKey, true)
	e.SetWriteGCMMode(true)

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

	err = e.DecryptSMCredentialsUsingMode(params, EncryptionModeGCM)
	require.NoError(t, err)
	assert.Equal(t, "gcm-client-id", params.ErsContext.SMOperatorCredentials.ClientID)
	assert.Equal(t, "gcm-client-secret", params.ErsContext.SMOperatorCredentials.ClientSecret)
}

func TestDecryptSMCredentialsUsingModeWithNilCredentials(t *testing.T) {
	secretKey := rand.String(32)
	e := NewEncrypter(secretKey, false)

	params := &internal.ProvisioningParameters{
		ErsContext: internal.ERSContext{
			SMOperatorCredentials: nil,
		},
	}

	err := e.DecryptSMCredentialsUsingMode(params, EncryptionModeCFB)
	require.NoError(t, err)
	assert.Nil(t, params.ErsContext.SMOperatorCredentials)
}

func TestDecryptSMCredentialsUsingModeWithEmptyCredentials(t *testing.T) {
	secretKey := rand.String(32)
	e := NewEncrypter(secretKey, false)

	params := &internal.ProvisioningParameters{
		ErsContext: internal.ERSContext{
			SMOperatorCredentials: &internal.ServiceManagerOperatorCredentials{
				ClientID:     "",
				ClientSecret: "",
			},
		},
	}

	err := e.DecryptSMCredentialsUsingMode(params, EncryptionModeCFB)
	require.NoError(t, err)
	assert.Equal(t, "", params.ErsContext.SMOperatorCredentials.ClientID)
	assert.Equal(t, "", params.ErsContext.SMOperatorCredentials.ClientSecret)
}

func TestDecryptKubeconfigUsingCFBMode(t *testing.T) {
	secretKey := rand.String(32)
	e := NewEncrypter(secretKey, false)

	params := &internal.ProvisioningParameters{
		Parameters: runtime.ProvisioningParametersDTO{
			Kubeconfig: "kubeconfig-cfb-content",
		},
	}

	err := e.EncryptKubeconfig(params)
	require.NoError(t, err)

	encryptedKubeconfig := params.Parameters.Kubeconfig
	assert.NotEqual(t, "kubeconfig-cfb-content", encryptedKubeconfig)

	err = e.DecryptKubeconfigUsingMode(params, EncryptionModeCFB)
	require.NoError(t, err)
	assert.Equal(t, "kubeconfig-cfb-content", params.Parameters.Kubeconfig)
}

func TestDecryptKubeconfigUsingGCMMode(t *testing.T) {
	secretKey := rand.String(32)
	e := NewEncrypter(secretKey, true)

	params := &internal.ProvisioningParameters{

		Parameters: runtime.ProvisioningParametersDTO{
			Kubeconfig: "kubeconfig-gcm-content",
		},
	}

	err := e.EncryptKubeconfig(params)
	require.NoError(t, err)

	encryptedKubeconfig := params.Parameters.Kubeconfig
	assert.NotEqual(t, "kubeconfig-gcm-content", encryptedKubeconfig)

	err = e.DecryptKubeconfigUsingMode(params, EncryptionModeGCM)
	require.NoError(t, err)
	assert.Equal(t, "kubeconfig-gcm-content", params.Parameters.Kubeconfig)
}

func TestDecryptKubeconfigUsingModeWithEmptyKubeconfig(t *testing.T) {
	secretKey := rand.String(32)
	e := NewEncrypter(secretKey, false)

	params := &internal.ProvisioningParameters{
		Parameters: runtime.ProvisioningParametersDTO{
			Kubeconfig: "",
		},
	}

	err := e.DecryptKubeconfigUsingMode(params, EncryptionModeCFB)
	require.NoError(t, err)
	assert.Equal(t, "", params.Parameters.Kubeconfig)
}

func TestDecryptSMCredentialsUsingModeWithDefaultFallbackToCFB(t *testing.T) {
	secretKey := rand.String(32)
	e := NewEncrypter(secretKey, false)

	params := &internal.ProvisioningParameters{
		ErsContext: internal.ERSContext{
			SMOperatorCredentials: &internal.ServiceManagerOperatorCredentials{
				ClientID:     "default-client-id",
				ClientSecret: "default-client-secret",
			},
		},
	}

	err := e.EncryptSMCredentials(params)
	require.NoError(t, err)

	err = e.DecryptSMCredentialsUsingMode(params, "unknown-mode")
	require.NoError(t, err)
	assert.Equal(t, "default-client-id", params.ErsContext.SMOperatorCredentials.ClientID)
	assert.Equal(t, "default-client-secret", params.ErsContext.SMOperatorCredentials.ClientSecret)
}

func TestDecryptKubeconfigUsingModeWithDefaultFallbackToCFB(t *testing.T) {
	secretKey := rand.String(32)
	e := NewEncrypter(secretKey, false)

	params := &internal.ProvisioningParameters{
		Parameters: runtime.ProvisioningParametersDTO{
			Kubeconfig: "default-kubeconfig",
		},
	}

	err := e.EncryptKubeconfig(params)
	require.NoError(t, err)

	err = e.DecryptKubeconfigUsingMode(params, "unknown-mode")
	require.NoError(t, err)
	assert.Equal(t, "default-kubeconfig", params.Parameters.Kubeconfig)
}

func TestEncryptAndDecryptWithDifferentModes(t *testing.T) {
	secretKey := rand.String(32)
	e := NewEncrypter(secretKey, false)

	data := []byte("mixed mode test data")

	cfbEncrypted, err := e.encryptCFB(data)
	require.NoError(t, err)

	e.SetWriteGCMMode(true)
	gcmEncrypted, err := e.encryptGCM(data)
	require.NoError(t, err)

	assert.NotEqual(t, cfbEncrypted, gcmEncrypted)

	cfbDecrypted, err := e.DecryptUsingMode(cfbEncrypted, EncryptionModeCFB)
	require.NoError(t, err)
	assert.Equal(t, data, cfbDecrypted)

	gcmDecrypted, err := e.DecryptUsingMode(gcmEncrypted, EncryptionModeGCM)
	require.NoError(t, err)
	assert.Equal(t, data, gcmDecrypted)
}

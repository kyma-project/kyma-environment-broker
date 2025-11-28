package storage

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/rand"
)

type testDto struct {
	Data string `json:"data"`
}

func TestNewEncrypterInCFBOnlyMode(t *testing.T) {
	secretKey := rand.String(32)
	encrypter := NewEncrypter(secretKey)

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

	e := NewEncrypter(secretKey)
	e.SetWriteGCMMode(true)

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

	e := NewEncrypter(secretKey)
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

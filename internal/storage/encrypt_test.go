package storage

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/rand"
)

func TestNewEncrypterInCFBOnlyMode(t *testing.T) {

	type testDto struct {
		Data string `json:"data"`
	}

	t.Run("encrypt json", func(t *testing.T) {
		secretKey := rand.String(32)

		e := NewEncrypter(secretKey)
		dto := testDto{
			Data: secretKey,
		}

		j, err := json.Marshal(&dto)
		require.NoError(t, err)

		cipherText, err := e.Encrypt(j)
		require.NoError(t, err)
		assert.NotEqual(t, j, cipherText)

		cipherText, err = e.decryptCFB(cipherText)
		require.NoError(t, err)
		assert.Equal(t, j, cipherText)

		_, err = e.decryptGCM(cipherText)
		require.Error(t, err)

		err = json.Unmarshal(cipherText, &dto)
		require.NoError(t, err)
	})

	t.Run("encrypt string", func(t *testing.T) {
		secretKey := rand.String(32)

		e := NewEncrypter(secretKey)
		dto := []byte("test")

		cipherText, err := e.Encrypt(dto)
		require.NoError(t, err)
		assert.NotEqual(t, dto, cipherText)

		cipherText, err = e.decryptCFB(cipherText)
		require.NoError(t, err)
		assert.Equal(t, dto, cipherText)

		_, err = e.decryptGCM(cipherText)
		require.Error(t, err)
	})

	t.Run("invalid key", func(t *testing.T) {
		secretKey := ""

		cipherText := NewEncrypter(secretKey)

		dto := testDto{
			Data: secretKey,
		}

		j, err := json.Marshal(&dto)
		require.NoError(t, err)

		_, err = cipherText.Encrypt(j)
		require.Error(t, err)
	})

}

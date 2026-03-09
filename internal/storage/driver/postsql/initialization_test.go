package postsql_test

import (
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/storage/postsql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitialization(t *testing.T) {

	t.Run("Should initialize database when schema not applied", func(t *testing.T) {
		storageCleanup, brokerStorage, err := GetStorageForDatabaseTests()
		require.NoError(t, err)
		require.NotNil(t, brokerStorage)
		defer func() {
			err := storageCleanup()
			assert.NoError(t, err)
		}()
	})

	t.Run("Should return error when failed to connect to the database with bad connection string", func(t *testing.T) {
		// given
		connString := "bad connection string"

		// when
		connection, err := postsql.InitializeDatabase(connString, 3)

		// then
		assert.Error(t, err)
		assert.Nil(t, connection)
	})
}

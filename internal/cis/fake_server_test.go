package cis

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCisFakeServer(t *testing.T) {
	srv, err := NewFakeServer()
	require.NoError(t, err)
	defer srv.Close()

	client := srv.Client()

	t.Run("should get a subaccount for the given ID", func(t *testing.T) {
		resp, err := client.Get(srv.URL + "/accounts/v1/technical/subaccounts/" + FakeSubaccountID1)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		data := make(map[string]interface{})
		err = json.NewDecoder(resp.Body).Decode(&data)
		require.NoError(t, err)

		assert.Equal(t, FakeSubaccountID1, data["guid"])
	})
}

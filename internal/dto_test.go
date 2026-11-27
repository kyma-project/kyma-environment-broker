package internal_test

import (
	"encoding/json"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdateInstanceERSContext_SMOperatorCredentials covers the field-level merge
// behaviour for SMOperatorCredentials. The contract: non-empty fields in `operation`
// overwrite the corresponding field in `instance`; empty fields are ignored.
func TestUpdateInstanceERSContext_SMOperatorCredentials(t *testing.T) {
	full := func() *internal.ServiceManagerOperatorCredentials {
		return &internal.ServiceManagerOperatorCredentials{
			ClientID:          "original-cid",
			ClientSecret:      "original-secret",
			ServiceManagerURL: "https://sm.original",
			URL:               "https://token.original",
			XSAppName:         "original-app",
		}
	}

	t.Run("nil credentials in operation leaves instance credentials unchanged", func(t *testing.T) {
		instance := internal.ERSContext{SMOperatorCredentials: full()}
		op := internal.ERSContext{SMOperatorCredentials: nil}

		result := internal.UpdateInstanceERSContext(instance, op)

		require.NotNil(t, result.SMOperatorCredentials)
		assert.Equal(t, "original-cid", result.SMOperatorCredentials.ClientID)
		assert.Equal(t, "original-secret", result.SMOperatorCredentials.ClientSecret)
		assert.Equal(t, "https://sm.original", result.SMOperatorCredentials.ServiceManagerURL)
		assert.Equal(t, "https://token.original", result.SMOperatorCredentials.URL)
		assert.Equal(t, "original-app", result.SMOperatorCredentials.XSAppName)
	})

	t.Run("full credentials in operation overwrite all fields", func(t *testing.T) {
		instance := internal.ERSContext{SMOperatorCredentials: full()}
		op := internal.ERSContext{
			SMOperatorCredentials: &internal.ServiceManagerOperatorCredentials{
				ClientID:          "new-cid",
				ClientSecret:      "new-secret",
				ServiceManagerURL: "https://sm.new",
				URL:               "https://token.new",
				XSAppName:         "new-app",
			},
		}

		result := internal.UpdateInstanceERSContext(instance, op)

		require.NotNil(t, result.SMOperatorCredentials)
		assert.Equal(t, "new-cid", result.SMOperatorCredentials.ClientID)
		assert.Equal(t, "new-secret", result.SMOperatorCredentials.ClientSecret)
		assert.Equal(t, "https://sm.new", result.SMOperatorCredentials.ServiceManagerURL)
		assert.Equal(t, "https://token.new", result.SMOperatorCredentials.URL)
		assert.Equal(t, "new-app", result.SMOperatorCredentials.XSAppName)
	})

	t.Run("only clientsecret provided — other fields preserved", func(t *testing.T) {
		instance := internal.ERSContext{SMOperatorCredentials: full()}
		op := internal.ERSContext{
			SMOperatorCredentials: &internal.ServiceManagerOperatorCredentials{
				ClientSecret: "rotated-secret",
				// all other fields are empty strings (JSON: {"clientsecret":"rotated-secret"})
			},
		}

		result := internal.UpdateInstanceERSContext(instance, op)

		require.NotNil(t, result.SMOperatorCredentials)
		assert.Equal(t, "original-cid", result.SMOperatorCredentials.ClientID, "clientid must be preserved")
		assert.Equal(t, "rotated-secret", result.SMOperatorCredentials.ClientSecret, "clientsecret must be updated")
		assert.Equal(t, "https://sm.original", result.SMOperatorCredentials.ServiceManagerURL, "sm_url must be preserved")
		assert.Equal(t, "https://token.original", result.SMOperatorCredentials.URL, "url must be preserved")
		assert.Equal(t, "original-app", result.SMOperatorCredentials.XSAppName, "xsappname must be preserved")
	})

	t.Run("only clientid provided — other fields preserved", func(t *testing.T) {
		instance := internal.ERSContext{SMOperatorCredentials: full()}
		op := internal.ERSContext{
			SMOperatorCredentials: &internal.ServiceManagerOperatorCredentials{
				ClientID: "new-cid",
			},
		}

		result := internal.UpdateInstanceERSContext(instance, op)

		assert.Equal(t, "new-cid", result.SMOperatorCredentials.ClientID)
		assert.Equal(t, "original-secret", result.SMOperatorCredentials.ClientSecret)
		assert.Equal(t, "https://sm.original", result.SMOperatorCredentials.ServiceManagerURL)
		assert.Equal(t, "https://token.original", result.SMOperatorCredentials.URL)
	})

	t.Run("only sm_url and url provided — clientid and clientsecret preserved", func(t *testing.T) {
		instance := internal.ERSContext{SMOperatorCredentials: full()}
		op := internal.ERSContext{
			SMOperatorCredentials: &internal.ServiceManagerOperatorCredentials{
				ServiceManagerURL: "https://sm.new",
				URL:               "https://token.new",
			},
		}

		result := internal.UpdateInstanceERSContext(instance, op)

		assert.Equal(t, "original-cid", result.SMOperatorCredentials.ClientID)
		assert.Equal(t, "original-secret", result.SMOperatorCredentials.ClientSecret)
		assert.Equal(t, "https://sm.new", result.SMOperatorCredentials.ServiceManagerURL)
		assert.Equal(t, "https://token.new", result.SMOperatorCredentials.URL)
	})

	t.Run("empty struct in operation (all fields empty string) leaves all existing fields unchanged", func(t *testing.T) {
		// This is what happens when JSON {"sm_operator_credentials":{}} is received.
		instance := internal.ERSContext{SMOperatorCredentials: full()}
		op := internal.ERSContext{
			SMOperatorCredentials: &internal.ServiceManagerOperatorCredentials{},
		}

		result := internal.UpdateInstanceERSContext(instance, op)

		require.NotNil(t, result.SMOperatorCredentials)
		assert.Equal(t, "original-cid", result.SMOperatorCredentials.ClientID)
		assert.Equal(t, "original-secret", result.SMOperatorCredentials.ClientSecret)
		assert.Equal(t, "https://sm.original", result.SMOperatorCredentials.ServiceManagerURL)
		assert.Equal(t, "https://token.original", result.SMOperatorCredentials.URL)
		assert.Equal(t, "original-app", result.SMOperatorCredentials.XSAppName)
	})

	t.Run("instance has nil credentials — partial operation creates new struct with provided fields only", func(t *testing.T) {
		// First-ever credential update on an instance provisioned without SM creds.
		instance := internal.ERSContext{SMOperatorCredentials: nil}
		op := internal.ERSContext{
			SMOperatorCredentials: &internal.ServiceManagerOperatorCredentials{
				ClientID:     "first-cid",
				ClientSecret: "first-secret",
			},
		}

		result := internal.UpdateInstanceERSContext(instance, op)

		require.NotNil(t, result.SMOperatorCredentials)
		assert.Equal(t, "first-cid", result.SMOperatorCredentials.ClientID)
		assert.Equal(t, "first-secret", result.SMOperatorCredentials.ClientSecret)
		assert.Equal(t, "", result.SMOperatorCredentials.ServiceManagerURL, "unprovided fields stay empty")
		assert.Equal(t, "", result.SMOperatorCredentials.URL)
	})

	t.Run("instance has nil credentials — full operation sets all fields", func(t *testing.T) {
		instance := internal.ERSContext{SMOperatorCredentials: nil}
		op := internal.ERSContext{
			SMOperatorCredentials: &internal.ServiceManagerOperatorCredentials{
				ClientID:          "cid",
				ClientSecret:      "secret",
				ServiceManagerURL: "https://sm.url",
				URL:               "https://token.url",
				XSAppName:         "app",
			},
		}

		result := internal.UpdateInstanceERSContext(instance, op)

		require.NotNil(t, result.SMOperatorCredentials)
		assert.Equal(t, "cid", result.SMOperatorCredentials.ClientID)
		assert.Equal(t, "secret", result.SMOperatorCredentials.ClientSecret)
		assert.Equal(t, "https://sm.url", result.SMOperatorCredentials.ServiceManagerURL)
		assert.Equal(t, "https://token.url", result.SMOperatorCredentials.URL)
		assert.Equal(t, "app", result.SMOperatorCredentials.XSAppName)
	})
}

// TestUpdateInstanceERSContext_NonCredentialFields verifies the other fields are
// still handled correctly after the SMOperatorCredentials refactor.
func TestUpdateInstanceERSContext_NonCredentialFields(t *testing.T) {
	t.Run("nil pointer fields in operation leave instance fields unchanged", func(t *testing.T) {
		commercial := "CUSTOMER"
		license := "TRIAL"
		origin := "origin-val"
		platform := "platform-val"
		region := "eu-west-1"
		instance := internal.ERSContext{
			CommercialModel: &commercial,
			LicenseType:     &license,
			Origin:          &origin,
			Platform:        &platform,
			Region:          &region,
		}
		op := internal.ERSContext{} // all nil

		result := internal.UpdateInstanceERSContext(instance, op)

		require.NotNil(t, result.CommercialModel)
		assert.Equal(t, "CUSTOMER", *result.CommercialModel)
		assert.Equal(t, "TRIAL", *result.LicenseType)
		assert.Equal(t, "origin-val", *result.Origin)
		assert.Equal(t, "platform-val", *result.Platform)
		assert.Equal(t, "eu-west-1", *result.Region)
	})

	t.Run("non-nil pointer fields in operation overwrite instance fields", func(t *testing.T) {
		commercial := "CUSTOMER"
		instance := internal.ERSContext{CommercialModel: &commercial}
		newCommercial := "PARTNER"
		op := internal.ERSContext{CommercialModel: &newCommercial}

		result := internal.UpdateInstanceERSContext(instance, op)

		require.NotNil(t, result.CommercialModel)
		assert.Equal(t, "PARTNER", *result.CommercialModel)
	})
}

// TestInheritMissingERSContext_SMOperatorCredentials verifies the inheritance
// logic is not broken by the SMOperatorCredentials change.
func TestInheritMissingERSContext_SMOperatorCredentials(t *testing.T) {
	t.Run("current nil creds inherited from previous", func(t *testing.T) {
		prev := &internal.ServiceManagerOperatorCredentials{ClientID: "prev-cid", ClientSecret: "prev-secret"}
		current := internal.ERSContext{SMOperatorCredentials: nil}
		previous := internal.ERSContext{SMOperatorCredentials: prev}

		result := internal.InheritMissingERSContext(current, previous)

		require.NotNil(t, result.SMOperatorCredentials)
		assert.Equal(t, "prev-cid", result.SMOperatorCredentials.ClientID)
	})

	t.Run("current non-nil creds not overwritten by previous", func(t *testing.T) {
		current := internal.ERSContext{
			SMOperatorCredentials: &internal.ServiceManagerOperatorCredentials{ClientID: "current-cid"},
		}
		previous := internal.ERSContext{
			SMOperatorCredentials: &internal.ServiceManagerOperatorCredentials{ClientID: "prev-cid"},
		}

		result := internal.InheritMissingERSContext(current, previous)

		assert.Equal(t, "current-cid", result.SMOperatorCredentials.ClientID)
	})
}

// TestERSContextERSUpdate_PartialCredentials verifies that a partial
// SMOperatorCredentials (non-nil pointer) still triggers ERSUpdate = true,
// so the update pipeline always runs when any credential field is sent.
func TestERSContextERSUpdate_PartialCredentials(t *testing.T) {
	ctx := internal.ERSContext{
		SMOperatorCredentials: &internal.ServiceManagerOperatorCredentials{
			ClientSecret: "rotated",
			// all other fields empty
		},
	}
	assert.True(t, ctx.ERSUpdate(), "partial credentials must still trigger ERSUpdate")
}

// TestUpdateInstanceERSContext_JSONRoundTrip simulates what happens when Go
// unmarshals {"clientsecret":"rotated"} into ERSContext — the pointer is non-nil
// and all other string fields are "". The merge must preserve existing DB values.
func TestUpdateInstanceERSContext_JSONRoundTrip(t *testing.T) {
	rawJSON := `{"sm_operator_credentials":{"clientsecret":"rotated-via-json"}}`

	var patchErsContext internal.ERSContext
	require.NoError(t, json.Unmarshal([]byte(rawJSON), &patchErsContext))

	// verify Go unmarshalled a non-nil struct with only clientsecret set
	require.NotNil(t, patchErsContext.SMOperatorCredentials)
	assert.Equal(t, "rotated-via-json", patchErsContext.SMOperatorCredentials.ClientSecret)
	assert.Equal(t, "", patchErsContext.SMOperatorCredentials.ClientID, "other fields must be empty string after JSON unmarshal")

	// simulate what processContext does: instance has full creds from DB
	instanceCreds := &internal.ServiceManagerOperatorCredentials{
		ClientID:          "db-cid",
		ClientSecret:      "db-old-secret",
		ServiceManagerURL: "https://sm.db",
		URL:               "https://token.db",
		XSAppName:         "db-app",
	}
	instanceCtx := internal.ERSContext{SMOperatorCredentials: instanceCreds}

	result := internal.UpdateInstanceERSContext(instanceCtx, patchErsContext)

	assert.Equal(t, "db-cid", result.SMOperatorCredentials.ClientID, "clientid must NOT be wiped by partial JSON")
	assert.Equal(t, "rotated-via-json", result.SMOperatorCredentials.ClientSecret)
	assert.Equal(t, "https://sm.db", result.SMOperatorCredentials.ServiceManagerURL)
	assert.Equal(t, "https://token.db", result.SMOperatorCredentials.URL)
	assert.Equal(t, "db-app", result.SMOperatorCredentials.XSAppName)
}

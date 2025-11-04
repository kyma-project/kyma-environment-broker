package machinesavailability

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/hyperscaler/rules"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/httputil"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestMachinesAvailabilityCBHandler(t *testing.T) {
	providerSpec, err := configuration.NewProviderSpecFromFile("testdata/providers.yaml")
	require.NoError(t, err)

	rulesService, err := rules.NewRulesServiceFromSlice([]string{"aws"}, sets.New("aws"), sets.New("aws"))
	require.NoError(t, err)

	fakeAWSClientFactory := fixture.NewFakeAWSClientFactory(map[string][]string{
		"m6i.large":    {"a", "b", "c", "d"},
		"m6i.xlarge":   {"a", "b", "c", "d"},
		"c7i.large":    {"a", "b"},
		"c7i.xlarge":   {"a", "b"},
		"g6.xlarge":    {"a", "b", "c"},
		"g6.2xlarge":   {"a", "b", "c"},
		"g4dn.xlarge":  {},
		"g4dn.2xlarge": {},
	}, nil)

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	handler := NewHandler(providerSpec, rulesService, fixture.CreateGardenerClient(), fakeAWSClientFactory, log)

	router := httputil.NewRouter()
	handler.AttachRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/oauth/v2/machines_availability", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Code)

	actualJSON := resp.Body.Bytes()

	expectedJSON, err := os.ReadFile("testdata/machines-availability.json")
	require.NoError(t, err)

	var actualData, expectedData interface{}
	require.NoError(t, json.Unmarshal(actualJSON, &actualData))
	require.NoError(t, json.Unmarshal(expectedJSON, &expectedData))

	assert.Equal(t, expectedData, actualData)
}

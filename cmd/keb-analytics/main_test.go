package main

import (
	"encoding/json"
	"testing"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/analytics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildFilteredStats_TrendsUsesSuppliedParamsWhenProvEmpty verifies that when the
// time-range filter results in no provisioning params (e.g. all instances were provisioned
// outside the selected window), trends are still built for the given trendParams rather
// than being silently empty. This is the AC6 bug: trends were derived from the
// time-filtered combined stats, producing an empty trendParams list.
func TestBuildFilteredStats_TrendsUsesSuppliedParamsWhenProvEmpty(t *testing.T) {
	// An instance was provisioned on day 1 (outside the 7-day window).
	provEvent := func(instanceID, day, machineType string) analytics.OpEvent {
		p := internal.ProvisioningParameters{
			Parameters: pkg.ProvisioningParametersDTO{MachineType: strPtr(machineType)},
		}
		raw, err := json.Marshal(p)
		require.NoError(t, err)
		return analytics.OpEvent{InstanceID: instanceID, CreatedAt: day, Type: "provision", RawParams: string(raw)}
	}

	opEvents := []analytics.OpEvent{
		provEvent("i1", "2024-01-01", "m6i.xlarge"),
	}

	// trendParams come from the full (unfiltered) combined stats — "machineType" is known.
	trendParams := []string{"machineType"}

	// provParams is empty (simulates a 7-day window where nothing was provisioned).
	resp := buildFilteredStats(nil, nil, opEvents, "", "", nil, nil, nil, trendParams)

	require.NotEmpty(t, resp.Trends, "trends must be non-empty when trendParams are supplied")
	assert.Equal(t, "machineType", resp.Trends[0].Parameter)
	require.NotEmpty(t, resp.Trends[0].Points, "trend points must be populated from op events")
}

func strPtr(s string) *string { return &s }

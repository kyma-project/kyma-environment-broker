package runtime

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdditionalWorkerNodePoolValidateLabels(t *testing.T) {
	pool := AdditionalWorkerNodePool{Name: "pool-1"}

	require.NoError(t, pool.ValidateLabels(map[string]string{"env": "prod"}, "pool-1"))
	require.NoError(t, pool.ValidateLabels(nil, "pool-1"))
	require.NoError(t, pool.ValidateLabels(map[string]string{}, "pool-1"))

	err := pool.ValidateLabels(map[string]string{"": "value"}, "pool-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "label key must not be empty")
}

func TestAdditionalWorkerNodePoolValidateAnnotations(t *testing.T) {
	pool := AdditionalWorkerNodePool{Name: "pool-1"}

	require.NoError(t, pool.ValidateAnnotations(map[string]string{"note": "test"}, "pool-1"))
	require.NoError(t, pool.ValidateAnnotations(nil, "pool-1"))
	require.NoError(t, pool.ValidateAnnotations(map[string]string{}, "pool-1"))

	err := pool.ValidateAnnotations(map[string]string{"": "value"}, "pool-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "annotation key must not be empty")
}

func TestAdditionalWorkerNodePoolUnmarshalJSON(t *testing.T) {
	base := `{"name":"pool-1","machineType":"m6i.large","haZones":true,"autoScalerMin":3,"autoScalerMax":20`

	tests := map[string]struct {
		json        string
		expectError bool
	}{
		"valid labels":             {json: base + `,"labels":{"a":"1","b":"2"}}`, expectError: false},
		"valid annotations":        {json: base + `,"annotations":{"a":"1","b":"2"}}`, expectError: false},
		"no labels or annotations": {json: base + `}`, expectError: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var pool AdditionalWorkerNodePool
			err := json.Unmarshal([]byte(tc.json), &pool)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCheckDuplicateWorkerNodePoolKeys(t *testing.T) {
	tests := map[string]struct {
		pools         string
		expectError   bool
		errorContains string
	}{
		"valid labels": {
			pools:       `[{"name":"pool-1","machineType":"m6i.large","haZones":true,"autoScalerMin":3,"autoScalerMax":20,"labels":{"a":"1","b":"2"}}]`,
			expectError: false,
		},
		"valid annotations": {
			pools:       `[{"name":"pool-1","machineType":"m6i.large","haZones":true,"autoScalerMin":3,"autoScalerMax":20,"annotations":{"a":"1","b":"2"}}]`,
			expectError: false,
		},
		"no labels or annotations": {
			pools:       `[{"name":"pool-1","machineType":"m6i.large","haZones":true,"autoScalerMin":3,"autoScalerMax":20}]`,
			expectError: false,
		},
		"duplicate label key": {
			pools:         `[{"name":"pool-1","machineType":"m6i.large","haZones":true,"autoScalerMin":3,"autoScalerMax":20,"labels":{"env":"prod","env":"dev"}}]`,
			expectError:   true,
			errorContains: `duplicate key "env" in labels`,
		},
		"duplicate annotation key": {
			pools:         `[{"name":"pool-1","machineType":"m6i.large","haZones":true,"autoScalerMin":3,"autoScalerMax":20,"annotations":{"cc":"123","cc":"456"}}]`,
			expectError:   true,
			errorContains: `duplicate key "cc" in annotations`,
		},
		"duplicate in second pool": {
			pools:         `[{"name":"pool-1","machineType":"m6i.large","haZones":true,"autoScalerMin":3,"autoScalerMax":20},{"name":"pool-2","machineType":"m6i.large","haZones":true,"autoScalerMin":3,"autoScalerMax":20,"labels":{"env":"prod","env":"dev"}}]`,
			expectError:   true,
			errorContains: `pool-2`,
		},
		"empty array": {
			pools:       `[]`,
			expectError: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := CheckDuplicateWorkerNodePoolKeys(json.RawMessage(tc.pools))
			if tc.expectError {
				require.Error(t, err)
				assert.True(t, strings.Contains(err.Error(), tc.errorContains),
					"expected error to contain %q, got: %s", tc.errorContains, err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

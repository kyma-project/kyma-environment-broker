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

	validKeys := []string{"app", "app.kubernetes.io/name", "a", "x-y.z_1", "MyName", "123-abc"}
	for _, k := range validKeys {
		require.NoError(t, pool.ValidateLabels(map[string]string{k: "v"}, "pool-1"), "expected valid key: %q", k)
	}

	invalidKeys := []string{"my.label.()", "key with space", "/no-name", ""}
	for _, k := range invalidKeys {
		err := pool.ValidateLabels(map[string]string{k: "v"}, "pool-1")
		require.Error(t, err, "expected invalid key: %q", k)
		assert.Contains(t, err.Error(), "label key", "key: %q", k)
	}

	require.NoError(t, pool.ValidateLabels(nil, "pool-1"))
	require.NoError(t, pool.ValidateLabels(map[string]string{}, "pool-1"))

	validValues := []string{"v1", "my-value", "", "a.b_c-d"}
	for _, v := range validValues {
		require.NoError(t, pool.ValidateLabels(map[string]string{"app": v}, "pool-1"), "expected valid value: %q", v)
	}

	invalidValues := []string{"value!", "value with space"}
	for _, v := range invalidValues {
		err := pool.ValidateLabels(map[string]string{"app": v}, "pool-1")
		require.Error(t, err, "expected invalid value: %q", v)
		assert.Contains(t, err.Error(), "label value", "value: %q", v)
	}
}

func TestAdditionalWorkerNodePoolValidateAnnotations(t *testing.T) {
	pool := AdditionalWorkerNodePool{Name: "pool-1"}

	require.NoError(t, pool.ValidateAnnotations(map[string]string{"note": "test"}, "pool-1"))
	require.NoError(t, pool.ValidateAnnotations(nil, "pool-1"))
	require.NoError(t, pool.ValidateAnnotations(map[string]string{}, "pool-1"))

	// annotation values are unrestricted
	require.NoError(t, pool.ValidateAnnotations(map[string]string{"app": "value with spaces!"}, "pool-1"))
	require.NoError(t, pool.ValidateAnnotations(map[string]string{"app": "value(with)parens"}, "pool-1"))

	// invalid keys
	invalidKeys := []string{"my.label.()", "key with space", "/no-name", ""}
	for _, k := range invalidKeys {
		err := pool.ValidateAnnotations(map[string]string{k: "any-value"}, "pool-1")
		require.Error(t, err, "expected invalid key: %q", k)
		assert.Contains(t, err.Error(), "annotation key", "key: %q", k)
	}
}

func TestAdditionalWorkerNodePoolValidateTaints(t *testing.T) {
	pool := AdditionalWorkerNodePool{Name: "pool-1"}

	validKeys := []string{"app", "app.kubernetes.io/name", "a", "x-y.z_1"}
	for _, k := range validKeys {
		err := pool.ValidateTaints([]TaintDTO{{Key: k, Effect: TaintEffectNoSchedule}}, "pool-1")
		require.NoError(t, err, "expected valid taint key: %q", k)
	}

	invalidKeys := []string{"my.label.()", "key with space", "/no-name", ""}
	for _, k := range invalidKeys {
		err := pool.ValidateTaints([]TaintDTO{{Key: k, Effect: TaintEffectNoSchedule}}, "pool-1")
		require.Error(t, err, "expected invalid taint key: %q", k)
		assert.Contains(t, err.Error(), "taint key", "key: %q", k)
	}

	validValues := []string{"v1", "my-value", "", "a.b_c-d"}
	for _, v := range validValues {
		err := pool.ValidateTaints([]TaintDTO{{Key: "app", Value: v, Effect: TaintEffectNoSchedule}}, "pool-1")
		require.NoError(t, err, "expected valid taint value: %q", v)
	}

	invalidValues := []string{"value!", "value with space"}
	for _, v := range invalidValues {
		err := pool.ValidateTaints([]TaintDTO{{Key: "app", Value: v, Effect: TaintEffectNoSchedule}}, "pool-1")
		require.Error(t, err, "expected invalid taint value: %q", v)
		assert.Contains(t, err.Error(), "taint value", "value: %q", v)
	}
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

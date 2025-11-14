package steps

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/fixture"
	"github.com/kyma-project/kyma-environment-broker/internal/ptr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

func TestInitKymaTemplate_Run(t *testing.T) {
	// given
	db := storage.NewMemoryStorage()
	operation := fixture.FixOperation("op-id", "inst-id", internal.OperationTypeProvision)
	err := db.Operations().InsertOperation(operation)
	require.NoError(t, err)

	svc := NewInitKymaTemplate(db.Operations(), &fakeConfigProvider{})

	// when
	op, backoff, err := svc.Run(operation, fixLogger())
	require.NoError(t, err)

	// then
	assert.Zero(t, backoff)
	assert.Equal(t, "kyma-system", op.KymaResourceNamespace)
	assert.NotEmptyf(t, op.KymaTemplate, "KymaTemplate should not be empty")
}

func TestInitKymaTemplate_ApplyChannelToTemplate(t *testing.T) {
	tests := []struct {
		name            string
		userChannel     *string
		expectedChannel string
		expectError     bool
	}{
		{
			name:            "applies user channel 'fast'",
			userChannel:     ptr.String("fast"),
			expectedChannel: "fast",
			expectError:     false,
		},
		{
			name:            "applies user channel 'regular'",
			userChannel:     ptr.String("regular"),
			expectedChannel: "regular",
			expectError:     false,
		},
		{
			name:            "keeps template default when no user channel",
			userChannel:     nil,
			expectedChannel: "stable", // from template
			expectError:     false,
		},
		{
			name:            "keeps template default when empty user channel",
			userChannel:     ptr.String(""),
			expectedChannel: "stable", // from template
			expectError:     false,
		},
		{
			name:        "fails with invalid channel",
			userChannel: ptr.String("invalid"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			db := storage.NewMemoryStorage()
			operation := fixture.FixOperation("op-id", "inst-id", internal.OperationTypeProvision)

			// Set up modules with channel
			if tt.userChannel != nil {
				operation.ProvisioningParameters.Parameters.Modules = &runtime.ModulesDTO{
					Channel: tt.userChannel,
					Default: ptr.Bool(true),
				}
			}

			err := db.Operations().InsertOperation(operation)
			require.NoError(t, err)

			svc := NewInitKymaTemplate(db.Operations(), &fakeConfigProvider{})

			// when
			op, backoff, err := svc.Run(operation, fixLogger())

			// then
			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Zero(t, backoff)
			assert.Equal(t, "kyma-system", op.KymaResourceNamespace)
			assert.NotEmpty(t, op.KymaTemplate)

			// Verify the channel was applied correctly
			decoded, decodeErr := DecodeKymaTemplate(op.KymaTemplate)
			require.NoError(t, decodeErr)

			actualChannel, found, getErr := unstructured.NestedString(decoded.Object, "spec", "channel")
			require.NoError(t, getErr)
			require.True(t, found, "channel should be present in template")
			assert.Equal(t, tt.expectedChannel, actualChannel)
		})
	}
}

func TestOverrideKymaModulesWithChannel(t *testing.T) {
	testCases := []struct {
		name                   string
		templateFile           string
		userChannel            string
		expectError            bool
		expectedGlobalChannel  string
		expectedModuleChannels map[string]string
	}{
		{
			name:                   "Regular channel - no modules",
			templateFile:           "kyma-channel-regular-no-modules.yaml",
			userChannel:            "regular",
			expectError:            false,
			expectedGlobalChannel:  "regular",
			expectedModuleChannels: map[string]string{},
		},
		{
			name:                   "Fast channel - no modules",
			templateFile:           "kyma-channel-fast-no-modules.yaml",
			userChannel:            "fast",
			expectError:            false,
			expectedGlobalChannel:  "fast",
			expectedModuleChannels: map[string]string{},
		},
		{
			name:                  "Regular channel with modules",
			templateFile:          "kyma-channel-regular-with-modules.yaml",
			userChannel:           "regular",
			expectError:           false,
			expectedGlobalChannel: "regular",
			expectedModuleChannels: map[string]string{
				"keda":         "regular",
				"btp-operator": "regular",
			},
		},
		{
			name:                  "Fast channel with modules",
			templateFile:          "kyma-channel-fast-with-modules.yaml",
			userChannel:           "fast",
			expectError:           false,
			expectedGlobalChannel: "fast",
			expectedModuleChannels: map[string]string{
				"keda":         "fast",
				"btp-operator": "fast",
			},
		},
		{
			name:                  "Mixed channels preserved",
			templateFile:          "kyma-mixed-channels-with-modules.yaml",
			userChannel:           "regular",
			expectError:           false,
			expectedGlobalChannel: "regular",
			expectedModuleChannels: map[string]string{
				"keda":         "regular",
				"btp-operator": "fast", // preserved from template
				"serverless":   "regular",
			},
		},
		{
			name:                  "Override default channel behavior",
			templateFile:          "kyma-override-default-channel.yaml",
			userChannel:           "fast",
			expectError:           false,
			expectedGlobalChannel: "fast",
			expectedModuleChannels: map[string]string{
				"keda":         "fast",    // inherits from global
				"btp-operator": "regular", // preserved from template
			},
		},
		{
			name:                  "Complex modules with channels",
			templateFile:          "kyma-complex-modules-with-channels.yaml",
			userChannel:           "regular",
			expectError:           false,
			expectedGlobalChannel: "regular",
			expectedModuleChannels: map[string]string{
				"keda":         "fast",    // preserved from template
				"btp-operator": "regular", // preserved from template
				"serverless":   "regular", // inherits from global
				"eventing":     "fast",    // preserved from template
			},
		},
		{
			name:                   "No user channel specified",
			templateFile:           "kyma-no-user-channel-no-modules.yaml",
			userChannel:            "",
			expectError:            false,
			expectedGlobalChannel:  "stable", // unchanged from template
			expectedModuleChannels: map[string]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			db := storage.NewMemoryStorage()

			// Load template
			templatePath := filepath.Join("testdata", "kymatemplate", tc.templateFile)
			templateContent, err := os.ReadFile(templatePath)
			require.NoError(t, err)

			// Create operation with user channel
			operation := fixture.FixOperation("op-id", "inst-id", internal.OperationTypeProvision)
			if tc.userChannel != "" {
				operation.ProvisioningParameters.Parameters.Modules = &runtime.ModulesDTO{
					Channel: ptr.String(tc.userChannel),
					Default: ptr.Bool(true),
				}
			}

			err = db.Operations().InsertOperation(operation)
			require.NoError(t, err)

			// Create fake config provider that returns our test template
			fakeProvider := &fakeTemplateConfigProvider{
				template: string(templateContent),
			}

			// Create and execute InitKymaTemplate step (this applies the channel)
			initStep := NewInitKymaTemplate(db.Operations(), fakeProvider)
			operation, _, err = initStep.Run(operation, fixLogger())
			if tc.expectError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Parse the final template and verify channels
			var kyma unstructured.Unstructured
			err = yaml.Unmarshal([]byte(operation.KymaTemplate), &kyma.Object)
			require.NoError(t, err)

			// Check global channel
			channel, found, err := unstructured.NestedString(kyma.Object, "spec", "channel")
			require.NoError(t, err)
			require.True(t, found)
			assert.Equal(t, tc.expectedGlobalChannel, channel)

			// Check module channels
			modules, found, err := unstructured.NestedSlice(kyma.Object, "spec", "modules")
			if len(tc.expectedModuleChannels) > 0 {
				require.NoError(t, err)
				require.True(t, found)

				moduleChannels := make(map[string]string)
				for _, m := range modules {
					module := m.(map[string]interface{})
					name := module["name"].(string)
					if ch, exists := module["channel"]; exists {
						moduleChannels[name] = ch.(string)
					} else {
						// Module inherits global channel
						moduleChannels[name] = tc.expectedGlobalChannel
					}
				}

				for expectedName, expectedChannel := range tc.expectedModuleChannels {
					actualChannel, exists := moduleChannels[expectedName]
					assert.True(t, exists, "Module %s should exist in final template", expectedName)
					assert.Equal(t, expectedChannel, actualChannel, "Module %s should have channel %s", expectedName, expectedChannel)
				}
			}
		})
	}
}

type fakeTemplateConfigProvider struct {
	template string
}

func (f *fakeTemplateConfigProvider) Provide(cfgKeyName string, cfgDestObj any) error {
	cfg, _ := cfgDestObj.(*internal.ConfigForPlan)
	cfg.KymaTemplate = f.template
	cfgDestObj = cfg
	return nil
}

type fakeConfigProvider struct{}

func (f fakeConfigProvider) Provide(cfgKeyName string, cfgDestObj any) error {
	cfg, _ := cfgDestObj.(*internal.ConfigForPlan)
	cfg.KymaTemplate = `apiVersion: operator.kyma-project.io/v1beta2
kind: Kyma
metadata:
    name: my-kyma
    namespace: kyma-system
spec:
    sync:
        strategy: secret
    channel: stable
    modules: []
`
	cfgDestObj = cfg
	return nil
}

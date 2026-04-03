package broker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func taintsSchemaExpected(rejectUnsupportedParameters bool) string {
	additionalPropertiesFragment := ""
	if rejectUnsupportedParameters {
		additionalPropertiesFragment = `"additionalProperties": false,`
	}

	return `{
		"type": "array",
		"description": "Specifies taints for the worker node pool. With the taints added, a node can repel sets of Pods with no matching tolerations.",
		"items": {
			"type": "object",
			` + additionalPropertiesFragment + `
			"_controlsOrder": ["key", "value", "effect"],
			"required": ["key", "effect"],
			"properties": {
				"key": {
					"type": "string",
					"description": "Specifies the taint key.",
					"minLength": 1
				},
				"value": {
					"type": "string",
					"description": "Specifies the taint value."
				},
				"effect": {
					"type": "string",
					"description": "Specifies the taint effect.",
					"enum": ["NoSchedule", "PreferNoSchedule", "NoExecute"],
					"_enumDisplayName": {
						"NoSchedule": "NoSchedule",
						"PreferNoSchedule": "PreferNoSchedule",
						"NoExecute": "NoExecute"
					}
				}
			}
		}
	}`
}

func TestNewTaintsSchema_WithAdditionalPropertiesAllowed(t *testing.T) {
	schema := NewTaintsSchema(false)

	marshaled := Marshal(schema)

	assert.JSONEq(t, taintsSchemaExpected(false), string(marshaled))
}

func TestNewTaintsSchema_WithAdditionalPropertiesRejected(t *testing.T) {
	schema := NewTaintsSchema(true)

	marshaled := Marshal(schema)

	assert.JSONEq(t, taintsSchemaExpected(true), string(marshaled))
}

func modulesSchemaExpected(additionalPropertiesOnListItems bool, defaultChannel string) string {
	additionalPropertiesFragment := ""
	if additionalPropertiesOnListItems {
		additionalPropertiesFragment = `"additionalProperties": false,`
	}

	channelDefault := `"` + defaultChannel + `"`

	return `{
		"type": "object",
		"description": "Use default modules or provide your custom list of modules. Provide an empty custom list of modules if you don't want any modules enabled.",
		"oneOf": [
			{
				"type": "object",
				"title": "Default",
				"description": "Default modules",
				"additionalProperties": false,
				"_controlsOrder": ["default", "channel"],
				"properties": {
					"default": {
						"type": "boolean",
						"title": "Use Default",
						"description": "Check the default modules in the <a href=https://help.sap.com/docs/btp/sap-business-technology-platform/kyma-modules?version=Cloud>default modules table</a>.",
						"default": true,
						"readOnly": true
					},
					"channel": {
						"type": "string",
						"title": "Default Module Channel",
						"description": "For the default modules, specifies your preferred default release channel: regular or fast. For details, see <a href=https://help.sap.com/docs/btp/sap-business-technology-platform/provisioning-and-update-parameters-in-kyma-environment?locale=en-US&q=IPv#modules>Modules</a>.",
						"default": ` + channelDefault + `,
						"enum": ["regular", "fast"],
						"_enumDisplayName": {
							"regular": "Regular - default version",
							"fast":    "Fast - latest version"
						}
					}
				}
			},
			{
				"type": "object",
				"title": "Custom",
				"description": "Define custom module list",
				"additionalProperties": false,
				"_controlsOrder": ["channel", "list"],
				"properties": {
					"channel": {
						"type": "string",
						"title": "Default Module Channel",
						"description": "Specifies your preferred release channel, regular or fast, for all modules in your custom list. You can change this setting for individual modules if needed. For details, see <a href=https://help.sap.com/docs/btp/sap-business-technology-platform/provisioning-and-update-parameters-in-kyma-environment?locale=en-US&q=IPv#modules>Modules</a>.",
						"default": ` + channelDefault + `,
						"enum": ["regular", "fast"],
						"_enumDisplayName": {
							"regular": "Regular - default version",
							"fast":    "Fast - latest version"
						}
					},
					"list": {
						"type": "array",
						"uniqueItems": true,
						"description": "Check a module technical name on this <a href=https://help.sap.com/docs/btp/sap-business-technology-platform/kyma-modules?version=Cloud>website</a>. You can only use a module technical name once. Provide an empty custom list of modules if you don't want any modules enabled.",
						"items": {
							"type": "object",
							` + additionalPropertiesFragment + `
							"_controlsOrder": ["name", "channel", "customResourcePolicy"],
							"properties": {
								"name": {
									"type": "string",
									"title": "Name",
									"description": "Check a module technical name on this <a href=https://help.sap.com/docs/btp/sap-business-technology-platform/kyma-modules?version=Cloud>website</a>. You can only use a module technical name once.",
									"minLength": 1
								},
								"channel": {
									"type": "string",
									"description": "Select your preferred release channel or leave this field empty. Overrides the Default Module Channel.",
									"default": "",
									"enum": ["", "regular", "fast"],
									"_enumDisplayName": {
										"": "",
										"regular": "Regular - default version",
										"fast":    "Fast - latest version"
									}
								},
								"customResourcePolicy": {
									"type": "string",
									"description": "Select your preferred CustomResourcePolicy setting or leave this field empty.",
									"default": "",
									"enum": ["", "CreateAndDelete", "Ignore"],
									"_enumDisplayName": {
										"":                "",
										"CreateAndDelete": "CreateAndDelete - default module resource is created or deleted.",
										"Ignore":          "Ignore - module resource is not created."
									}
								}
							}
						}
					}
				}
			}
		]
	}`
}

func TestNewModulesSchema_AdditionalPropertiesAllowed_EmptyChannel(t *testing.T) {
	schema := NewModulesSchema(false, "")

	marshaled := Marshal(schema)

	assert.JSONEq(t, modulesSchemaExpected(false, ""), string(marshaled))
}

func TestNewModulesSchema_AdditionalPropertiesAllowed_RegularChannel(t *testing.T) {
	schema := NewModulesSchema(false, "regular")

	marshaled := Marshal(schema)

	assert.JSONEq(t, modulesSchemaExpected(false, "regular"), string(marshaled))
}

func TestNewModulesSchema_AdditionalPropertiesAllowed_FastChannel(t *testing.T) {
	schema := NewModulesSchema(false, "fast")

	marshaled := Marshal(schema)

	assert.JSONEq(t, modulesSchemaExpected(false, "fast"), string(marshaled))
}

func TestNewModulesSchema_AdditionalPropertiesRejected_EmptyChannel(t *testing.T) {
	schema := NewModulesSchema(true, "")

	marshaled := Marshal(schema)

	assert.JSONEq(t, modulesSchemaExpected(true, ""), string(marshaled))
}

func TestNewModulesSchema_AdditionalPropertiesRejected_RegularChannel(t *testing.T) {
	schema := NewModulesSchema(true, "regular")

	marshaled := Marshal(schema)

	assert.JSONEq(t, modulesSchemaExpected(true, "regular"), string(marshaled))
}

func TestNewModulesSchema_AdditionalPropertiesRejected_FastChannel(t *testing.T) {
	schema := NewModulesSchema(true, "fast")

	marshaled := Marshal(schema)

	assert.JSONEq(t, modulesSchemaExpected(true, "fast"), string(marshaled))
}

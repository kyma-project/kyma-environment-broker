package broker

import (
	"encoding/json"
	"fmt"
	"strings"

	pkg "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/kyma-project/kyma-environment-broker/internal/networking"
	"github.com/kyma-project/kyma-environment-broker/internal/provider/configuration"
)

const (
	nonHAAutoscalerMinMinimumValue = 0
	nonHAAutoscalerMaxMinimumValue = 1
	autoscalerMaximumValue         = 300
	autoscalerMaxDefaultValue      = 20
)

type RootSchema struct {
	Schema string `json:"$schema"`
	Type
	Properties interface{} `json:"properties"`
	Required   []string    `json:"required,omitempty"`

	// Specified to true enables form view on website
	ShowFormView bool `json:"_show_form_view"`
	// Specifies in what order properties will be displayed on the form
	ControlsOrder []string `json:"_controlsOrder"`
	// Specified to true loads current instance configuration into the update instance schema
	LoadCurrentConfig bool `json:"_load_current_config,omitempty"`
}

type ProvisioningProperties struct {
	UpdateProperties

	ShootName            *Type           `json:"shootName,omitempty"`
	ShootDomain          *Type           `json:"shootDomain,omitempty"`
	Region               *Type           `json:"region,omitempty"`
	Networking           *NetworkingType `json:"networking,omitempty"`
	Modules              *Modules        `json:"modules,omitempty"`
	ColocateControlPlane *Type           `json:"colocateControlPlane,omitempty"`
}

type UpdateProperties struct {
	Name                      NameType                       `json:"name"`
	Kubeconfig                *Type                          `json:"kubeconfig,omitempty"`
	AutoScalerMin             *AutoscalerType                `json:"autoScalerMin,omitempty"`
	AutoScalerMax             *AutoscalerType                `json:"autoScalerMax,omitempty"`
	OIDC                      *OIDCs                         `json:"oidc,omitempty"`
	Administrators            *Type                          `json:"administrators,omitempty"`
	MachineType               *Type                          `json:"machineType,omitempty"`
	AdditionalWorkerNodePools *AdditionalWorkerNodePoolsType `json:"additionalWorkerNodePools,omitempty"`
	IngressFiltering          *Type                          `json:"ingressFiltering,omitempty"`
}

type NetworkingProperties struct {
	Nodes     Type  `json:"nodes"`
	Services  Type  `json:"services"`
	Pods      Type  `json:"pods"`
	DualStack *Type `json:"dualStack,omitempty"`
}

type NetworkingType struct {
	Type
	Properties NetworkingProperties `json:"properties"`
	Required   []string             `json:"required"`
}

type OIDCProperties struct {
	ClientID         Type `json:"clientID"`
	GroupsClaim      Type `json:"groupsClaim"`
	IssuerURL        Type `json:"issuerURL"`
	SigningAlgs      Type `json:"signingAlgs"`
	UsernameClaim    Type `json:"usernameClaim"`
	UsernamePrefix   Type `json:"usernamePrefix"`
	EncodedJwksArray Type `json:"encodedJwksArray,omitzero"`
}

type OIDCPropertiesExpanded struct {
	OIDCProperties
	RequiredClaims Type `json:"requiredClaims"`
	GroupsPrefix   Type `json:"groupsPrefix"`
}

type OIDCType struct {
	Type
	Properties    OIDCProperties `json:"properties"`
	Required      []string       `json:"required"`
	ControlsOrder []string       `json:"_controlsOrder,omitempty"`
}

type OIDCTypeExpanded struct {
	Type
	Properties    OIDCPropertiesExpanded `json:"properties"`
	Required      []string               `json:"required"`
	ControlsOrder []string               `json:"_controlsOrder,omitempty"`
}

type Type struct {
	Type        string `json:"type"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Minimum     int    `json:"minimum,omitempty"`
	Maximum     int    `json:"maximum,omitempty"`
	MinLength   int    `json:"minLength,omitempty"`
	MaxLength   int    `json:"maxLength,omitempty"`
	MinItems    int    `json:"minItems,omitempty"`
	MaxItems    int    `json:"maxItems,omitempty"`

	// Regex pattern to match against string type of fields.
	// If not specified for strings user can pass empty string with whitespaces only.
	Pattern              string            `json:"pattern,omitempty"`
	Default              interface{}       `json:"default,omitempty"`
	Example              interface{}       `json:"example,omitempty"`
	Enum                 []interface{}     `json:"enum,omitempty"`
	EnumDisplayName      map[string]string `json:"_enumDisplayName,omitempty"`
	Items                *Type             `json:"items,omitempty"`
	AdditionalItems      interface{}       `json:"additionalItems,omitempty"`
	UniqueItems          interface{}       `json:"uniqueItems,omitempty"`
	ReadOnly             interface{}       `json:"readOnly,omitempty"`
	AdditionalProperties interface{}       `json:"additionalProperties,omitempty"`
}

type AutoscalerType struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Minimum     int         `json:"minimum"`
	Maximum     int         `json:"maximum"`
	Default     interface{} `json:"default,omitempty"`
}

type NameType struct {
	Type
	BTPdefaultTemplate *BTPdefaultTemplate `json:"_BTPdefaultTemplate,omitempty"`
}

type BTPdefaultTemplate struct {
	Elements  []string `json:"elements,omitempty"`
	Separator string   `json:"separator,omitempty"`
}

type Modules struct {
	Type
	ControlsOrder []string      `json:"_controlsOrder,omitempty"`
	OneOf         []interface{} `json:"oneOf,omitempty"`
}

type ModulesDefault struct {
	Type
	Properties ModulesDefaultProperties `json:"properties,omitempty"`
}

type ModulesDefaultProperties struct {
	Default Type `json:"default,omitempty"`
}

type ModulesCustom struct {
	Type
	Properties ModulesCustomProperties `json:"properties,omitempty"`
}

type ModulesCustomProperties struct {
	List ModulesCustomList `json:"list,omitempty"`
}

type ModulesCustomList struct {
	Type
	Items ModulesCustomListItems `json:"items,omitempty"`
}

type ModulesCustomListItems struct {
	Type
	ControlsOrder []string                         `json:"_controlsOrder,omitempty"`
	Properties    ModulesCustomListItemsProperties `json:"properties,omitempty"`
}

type ModulesCustomListItemsProperties struct {
	Name                 Type `json:"name,omitempty"`
	Channel              Type `json:"channel,omitempty"`
	CustomResourcePolicy Type `json:"customResourcePolicy,omitempty"`
}

type AdditionalWorkerNodePoolsType struct {
	Type
	Items AdditionalWorkerNodePoolsItems `json:"items,omitempty"`
}

type AdditionalWorkerNodePoolsItems struct {
	Type
	ControlsOrder []string                                 `json:"_controlsOrder,omitempty"`
	Required      []string                                 `json:"required,omitempty"`
	Properties    AdditionalWorkerNodePoolsItemsProperties `json:"properties,omitempty"`
}

type AdditionalWorkerNodePoolsItemsProperties struct {
	Name          Type           `json:"name,omitempty"`
	MachineType   Type           `json:"machineType,omitempty"`
	HAZones       *Type          `json:"haZones,omitempty"`
	AutoScalerMin AutoscalerType `json:"autoScalerMin,omitempty"`
	AutoScalerMax AutoscalerType `json:"autoScalerMax,omitempty"`
}

type OIDCs struct {
	Type
	ControlsOrder []string      `json:"_controlsOrder,omitempty"`
	OneOf         []interface{} `json:"oneOf,omitempty"`
}
type AdditionalOIDC struct {
	Type
	Properties AdditionalOIDCProperties `json:"properties,omitempty"`
}

type AdditionalOIDCProperties struct {
	List AdditionalOIDCList `json:"list,omitempty"`
}

type AdditionalOIDCList struct {
	Type
	Items AdditionalOIDCListItems `json:"items,omitempty"`
}

type AdditionalOIDCListItems struct {
	Type
	ControlsOrder []string               `json:"_controlsOrder,omitempty"`
	Properties    OIDCPropertiesExpanded `json:"properties,omitempty"`
	Required      []string               `json:"required,omitempty"`
}

func NewMultipleOIDCSchema(defaultOIDCConfig *pkg.OIDCConfigDTO, update, rejectUnsupportedParameters bool) *OIDCs {
	if defaultOIDCConfig == nil {
		defaultOIDCConfig = &pkg.OIDCConfigDTO{}
	}

	var defaultOIDCListEntry map[string]interface{}
	if update {
		defaultOIDCListEntry = map[string]interface{}{
			"clientID":       "",
			"issuerURL":      "",
			"groupsClaim":    "",
			"signingAlgs":    []interface{}{},
			"usernameClaim":  "",
			"usernamePrefix": "",
			"groupsPrefix":   "",
			"requiredClaims": []interface{}{},
		}
	} else {
		defaultOIDCListEntry = map[string]interface{}{
			"clientID":       defaultOIDCConfig.ClientID,
			"issuerURL":      defaultOIDCConfig.IssuerURL,
			"groupsClaim":    defaultOIDCConfig.GroupsClaim,
			"signingAlgs":    defaultOIDCConfig.SigningAlgs,
			"usernameClaim":  defaultOIDCConfig.UsernameClaim,
			"usernamePrefix": defaultOIDCConfig.UsernamePrefix,
			"groupsPrefix":   defaultOIDCConfig.GroupsPrefix,
			"requiredClaims": []interface{}{},
		}
	}

	oidcListDescription := "Specifies the list of OIDC configurations. Besides the default OIDC configuration, you can add multiple custom OIDC configurations. Leave the list empty to not use any OIDC configuration."
	if update {
		oidcListDescription += " To use the default configuration values, see the documentation. To switch the OIDC configuration from a single oidc object to an oidc list, rewrite your values."
	}

	oidcDescription := "OIDC configuration. The list-based configuration is recommended. The object-based configuration is provided for backward compatibility."
	if !update {
		oidcDescription += " The object-based configuration inputs are still writable, but only from the JSON view."
	}

	OIDCs := &OIDCs{
		Type: Type{
			Type:        "object",
			Description: oidcDescription,
		},
		OneOf: []any{
			AdditionalOIDC{
				Type: Type{
					Type:                 "object",
					Title:                "List",
					Description:          "OIDC configuration list",
					AdditionalProperties: false,
				},
				Properties: AdditionalOIDCProperties{
					AdditionalOIDCList{
						Type: Type{
							Type:        "array",
							UniqueItems: true,
							Description: oidcListDescription,
							Default: []interface{}{
								defaultOIDCListEntry,
							},
						},
						Items: AdditionalOIDCListItems{
							ControlsOrder: []string{"clientID", "groupsClaim", "issuerURL", "signingAlgs", "usernameClaim", "usernamePrefix", "groupsPrefix", "requiredClaims"}, Type: Type{
								Type:                 "object",
								AdditionalProperties: false,
							},
							Properties: OIDCPropertiesExpanded{
								OIDCProperties: OIDCProperties{
									ClientID:       Type{Type: "string", MinLength: 1, Description: "The client ID for the OpenID Connect client."},
									IssuerURL:      Type{Type: "string", MinLength: 1, Description: "The URL of the OpenID issuer, only HTTPS scheme will be accepted."},
									GroupsClaim:    Type{Type: "string", MinLength: 1, Default: defaultOIDCConfig.GroupsClaim, Description: "If provided, the name of a custom OpenID Connect claim for specifying user groups."},
									UsernameClaim:  Type{Type: "string", MinLength: 1, Default: defaultOIDCConfig.UsernameClaim, Description: "The OpenID claim to use as the user name."},
									UsernamePrefix: Type{Type: "string", MinLength: 1, Default: defaultOIDCConfig.UsernamePrefix, Description: "If provided, all usernames are prefixed with this value. If not provided, username claims other than 'email' are prefixed by the issuer URL to avoid clashes. To skip any prefixing, provide the value '-' (dash character without additional characters)."},
									SigningAlgs: Type{
										Type:     "array",
										MinItems: 1,
										Items: &Type{
											Type: "string",
										},
										Default:     defaultOIDCConfig.SigningAlgs,
										Description: "A comma-separated list of allowed JOSE asymmetric signing algorithms, for example, RS256, ES256.",
									},
								},
								GroupsPrefix: Type{Type: "string", MinLength: 1, Default: defaultOIDCConfig.GroupsPrefix, Description: "If specified, causes claims mapping to group names to be prefixed with the current value. The 'oidc:' value results in groups like 'oidc:engineering' and 'oidc:marketing'. The value '-' (dash character without additional characters) can be used to disable all prefixing."},
								RequiredClaims: Type{
									Type: "array",
									Items: &Type{
										Type:    "string",
										Pattern: "^[^=]+=[^=]+$",
									},
									Description: "A list of key=value pairs that describes a required claim in the ID Token. If set, the claim is verified to be present in the ID Token with a matching value.",
								},
							},
							Required: []string{"clientID", "issuerURL", "groupsClaim", "usernameClaim", "usernamePrefix", "signingAlgs", "groupsPrefix"},
						},
					},
				},
			},
			OIDCTypeExpanded{
				ControlsOrder: []string{"clientID", "groupsClaim", "issuerURL", "signingAlgs", "usernameClaim", "usernamePrefix", "groupsPrefix", "requiredClaims"},
				Type: Type{
					Type:        "object",
					Title:       "Object (not recommended)",
					Description: "Legacy OIDC configuration",
				},
				Properties: OIDCPropertiesExpanded{
					OIDCProperties: OIDCProperties{
						ClientID:       Type{Type: "string", ReadOnly: !update, Description: "The client ID for the OpenID Connect client."},
						IssuerURL:      Type{Type: "string", ReadOnly: !update, Description: "The URL of the OpenID issuer, only HTTPS scheme will be accepted."},
						GroupsClaim:    Type{Type: "string", ReadOnly: !update, Description: "If provided, the name of a custom OpenID Connect claim for specifying user groups."},
						UsernameClaim:  Type{Type: "string", ReadOnly: !update, Description: "The OpenID claim to use as the user name."},
						UsernamePrefix: Type{Type: "string", ReadOnly: !update, Description: "If provided, all usernames are prefixed with this value. If not provided, username claims other than 'email' are prefixed by the issuer URL to avoid clashes. To skip any prefixing, provide the value '-' (dash character without additional characters)."},
						SigningAlgs: Type{
							Type: "array",
							Items: &Type{
								Type: "string",
							},
							ReadOnly:    !update,
							Description: "A comma-separated list of allowed JOSE asymmetric signing algorithms, for example, RS256, ES256.",
						},
					},
					GroupsPrefix: Type{Type: "string", ReadOnly: !update, Description: "If specified, causes claims mapping to group names to be prefixed with the current value. The 'oidc:' value results in groups like 'oidc:engineering' and 'oidc:marketing'. The value '-' (dash character without additional characters) can be used to disable all prefixing."},
					RequiredClaims: Type{
						Type: "array",
						Items: &Type{
							Type:    "string",
							Pattern: "^([^=]+=[^=]+|-)$",
						},
						ReadOnly:    !update,
						Description: "A list of key=value pairs that describes a required claim in the ID Token. If set, the claim is verified to be present in the ID Token with a matching value. To remove the previously set claims, use the value '-' (dash character without additional characters).",
					},
				},
				Required: []string{"clientID", "issuerURL"},
			},
		},
	}
	if rejectUnsupportedParameters {
		if oidcTypeExpanded, ok := OIDCs.OneOf[1].(OIDCTypeExpanded); ok {
			oidcTypeExpanded.Type.AdditionalProperties = false
			OIDCs.OneOf[1] = oidcTypeExpanded
		}
	}
	if additionalOidc, ok := OIDCs.OneOf[0].(AdditionalOIDC); ok {
		additionalOidc.Properties.List.Items.Properties.EncodedJwksArray = Type{Type: "string", Description: "The JWKS array encoded in base64. Leave empty to not use it or to remove the previously set value."}
		additionalOidc.Properties.List.Items.ControlsOrder = []string{"clientID", "groupsClaim", "issuerURL", "signingAlgs", "usernameClaim", "usernamePrefix", "groupsPrefix", "requiredClaims", "encodedJwksArray"}
		if update {
			additionalOidc.Properties.List.Default = []interface{}{
				map[string]interface{}{
					"clientID":         "",
					"issuerURL":        "",
					"groupsClaim":      "",
					"signingAlgs":      []interface{}{},
					"usernameClaim":    "",
					"usernamePrefix":   "",
					"groupsPrefix":     "",
					"requiredClaims":   []interface{}{},
					"encodedJwksArray": "",
				},
			}
		} else {
			additionalOidc.Properties.List.Default = []interface{}{
				map[string]interface{}{
					"clientID":         defaultOIDCConfig.ClientID,
					"issuerURL":        defaultOIDCConfig.IssuerURL,
					"groupsClaim":      defaultOIDCConfig.GroupsClaim,
					"signingAlgs":      defaultOIDCConfig.SigningAlgs,
					"usernameClaim":    defaultOIDCConfig.UsernameClaim,
					"usernamePrefix":   defaultOIDCConfig.UsernamePrefix,
					"groupsPrefix":     defaultOIDCConfig.GroupsPrefix,
					"requiredClaims":   []interface{}{},
					"encodedJwksArray": "",
				},
			}
		}
		OIDCs.OneOf[0] = additionalOidc
	}
	if oidcTypeExpanded, ok := OIDCs.OneOf[1].(OIDCTypeExpanded); ok {
		oidcTypeExpanded.Properties.EncodedJwksArray = Type{Type: "string", ReadOnly: !update, Description: "The JWKS array encoded in base64. To remove the previously set value, enter the value '-' (dash character without additional characters)."}
		oidcTypeExpanded.ControlsOrder = []string{"clientID", "groupsClaim", "issuerURL", "signingAlgs", "usernameClaim", "usernamePrefix", "groupsPrefix", "requiredClaims", "encodedJwksArray"}
		OIDCs.OneOf[1] = oidcTypeExpanded
	}
	return OIDCs
}

func NewOIDCSchema(rejectUnsupportedParameters bool) *OIDCType {
	OIDCType := &OIDCType{
		ControlsOrder: []string{"clientID", "groupsClaim", "issuerURL", "signingAlgs", "usernameClaim", "usernamePrefix"},
		Type:          Type{Type: "object", Description: "OIDC configuration"},
		Properties: OIDCProperties{
			ClientID:       Type{Type: "string", Description: "The client ID for the OpenID Connect client."},
			IssuerURL:      Type{Type: "string", Description: "The URL of the OpenID issuer, only HTTPS scheme will be accepted."},
			GroupsClaim:    Type{Type: "string", Description: "If provided, the name of a custom OpenID Connect claim for specifying user groups."},
			UsernameClaim:  Type{Type: "string", Description: "The OpenID claim to use as the user name."},
			UsernamePrefix: Type{Type: "string", Description: "If provided, all usernames are prefixed with this value. If not provided, username claims other than 'email' are prefixed by the issuer URL to avoid clashes. To skip any prefixing, provide the value '-' (dash character without additional characters)."},
			SigningAlgs: Type{
				Type: "array",
				Items: &Type{
					Type: "string",
				},
				Description: "A comma-separated list of allowed JOSE asymmetric signing algorithms, for example, RS256, ES256.",
			},
		},
		Required: []string{"clientID", "issuerURL"},
	}
	if rejectUnsupportedParameters {
		OIDCType.Type.AdditionalProperties = false
	}
	OIDCType.Properties.EncodedJwksArray = Type{Type: "string", Description: "JWKS array encoded in base64. To remove a previously set value, enter a single dash character '-'."}
	OIDCType.ControlsOrder = []string{"clientID", "groupsClaim", "issuerURL", "signingAlgs", "usernameClaim", "usernamePrefix", "encodedJwksArray"}
	return OIDCType
}

func NewModulesSchema(rejectUnsupportedParameters bool) *Modules {
	modules := &Modules{
		Type: Type{
			Type:        "object",
			Description: "Use default modules or provide your custom list of modules. Provide an empty custom list of modules if you don’t want any modules enabled.",
		},
		ControlsOrder: []string{"default", "list"},
		OneOf: []any{
			ModulesDefault{
				Type: Type{
					Type:                 "object",
					Title:                "Default",
					Description:          "Default modules",
					AdditionalProperties: false,
				},
				Properties: ModulesDefaultProperties{
					Type{
						Type:        "boolean",
						Title:       "Use Default",
						Description: "Check the default modules in the <a href=https://help.sap.com/docs/btp/sap-business-technology-platform/kyma-modules?version=Cloud>default modules table</a>.",
						Default:     true,
						ReadOnly:    true,
					},
				},
			},
			ModulesCustom{
				Type: Type{
					Type:                 "object",
					Title:                "Custom",
					Description:          "Define custom module list",
					AdditionalProperties: false,
				},
				Properties: ModulesCustomProperties{
					ModulesCustomList{
						Type: Type{
							Type:        "array",
							UniqueItems: true,
							Description: "Check a module technical name on this <a href=https://help.sap.com/docs/btp/sap-business-technology-platform/kyma-modules?version=Cloud>website</a>. You can only use a module technical name once. Provide an empty custom list of modules if you don’t want any modules enabled."},
						Items: ModulesCustomListItems{
							ControlsOrder: []string{"name", "channel", "customResourcePolicy"},
							Type: Type{
								Type: "object",
							},
							Properties: ModulesCustomListItemsProperties{
								Name: Type{
									Type:        "string",
									Title:       "Name",
									MinLength:   1,
									Description: "Check a module technical name on this <a href=https://help.sap.com/docs/btp/sap-business-technology-platform/kyma-modules?version=Cloud>website</a>. You can only use a module technical name once.",
								},
								Channel: Type{
									Type:        "string",
									Default:     "",
									Description: "Select your preferred release channel or leave this field empty.",
									Enum:        ToInterfaceSlice([]string{"", "regular", "fast"}),
									EnumDisplayName: map[string]string{
										"":        "",
										"regular": "Regular - default version",
										"fast":    "Fast - latest version",
									},
								},
								CustomResourcePolicy: Type{
									Type:        "string",
									Description: "Select your preferred CustomResourcePolicy setting or leave this field empty.",
									Default:     "",
									Enum:        ToInterfaceSlice([]string{"", "CreateAndDelete", "Ignore"}),
									EnumDisplayName: map[string]string{
										"":                "",
										"CreateAndDelete": "CreateAndDelete - default module resource is created or deleted.",
										"Ignore":          "Ignore - module resource is not created.",
									},
								},
							},
						},
					}},
			}},
	}
	if rejectUnsupportedParameters {
		if modulesCustom, ok := modules.OneOf[1].(ModulesCustom); ok {
			modulesCustom.Properties.List.Items.AdditionalProperties = false
			modules.OneOf[1] = modulesCustom
		}
	}
	return modules
}

func NameProperty(update bool) NameType {
	nameType := NameType{
		Type: Type{
			Type:  "string",
			Title: "Cluster Name",
			// Allows for all alphanumeric characters and '-'
			Pattern:   "^[a-zA-Z0-9-]*$",
			MinLength: 1,
			MaxLength: 64,
		},
	}

	if !update {
		nameType.BTPdefaultTemplate = &BTPdefaultTemplate{
			Elements: []string{"saSubdomain"},
		}
	}

	return nameType
}

func KubeconfigProperty() *Type {
	return &Type{
		Type:  "string",
		Title: "Kubeconfig contents",
	}
}

func ShootNameProperty() *Type {
	return &Type{
		Type:      "string",
		Title:     "Shoot name",
		Pattern:   "^[a-zA-Z0-9-]*$",
		MinLength: 1,
	}
}

func ShootDomainProperty() *Type {
	return &Type{
		Type:      "string",
		Title:     "Shoot domain",
		Pattern:   "^[a-zA-Z0-9-\\.]*$",
		MinLength: 1,
	}
}

func ColocateControlPlaneProperty() *Type {
	return &Type{
		Type:        "boolean",
		Title:       "Colocate control plane and worker nodes in the same region",
		Default:     false,
		Description: "If set to true, the control plane is placed in the same region as the selected region from the Region field. Check regions supporting the feature on this <a href=https://help.sap.com/docs/btp/sap-business-technology-platform/provisioning-and-update-parameters-in-kyma-environment?locale=en-US#region*>website</a>. The provisioning process fails if the control plane cannot be colocated in the region.",
	}
}

func IngressFilteringProperty() *Type {
	return &Type{
		Type:        "boolean",
		Title:       "Enable ingress geo-blocking",
		Default:     false,
		Description: "If set to true, ingress traffic from embargoed countries is blocked.",
	}
}

// NewProvisioningProperties creates a new properties for different plans
// Note that the order of properties will be the same in the form on the website
func NewProvisioningProperties(machineTypesDisplay, additionalMachineTypesDisplay, regionsDisplay map[string]string, machineTypes, additionalMachineTypes, regions []string, update, rejectUnsupportedParameters bool, providerSpec *configuration.ProviderSpec, cloudProvider pkg.CloudProvider, dualStackDocsURL string) ProvisioningProperties {

	properties := ProvisioningProperties{
		UpdateProperties: UpdateProperties{
			Name: NameProperty(update),
			AutoScalerMin: &AutoscalerType{
				Type:        "integer",
				Minimum:     pkg.HAAutoscalerMinimumValue,
				Maximum:     autoscalerMaximumValue,
				Default:     pkg.HAAutoscalerMinimumValue,
				Description: "Specifies the minimum number of virtual machines to create",
			},
			AutoScalerMax: &AutoscalerType{
				Type:        "integer",
				Minimum:     pkg.HAAutoscalerMinimumValue,
				Maximum:     autoscalerMaximumValue,
				Default:     autoscalerMaxDefaultValue,
				Description: "Specifies the maximum number of virtual machines to create",
			},
			MachineType: &Type{
				Type:            "string",
				Enum:            ToInterfaceSlice(machineTypes),
				EnumDisplayName: machineTypesDisplay,
				Description:     "Specifies the type of the virtual machine.",
			},
			AdditionalWorkerNodePools: NewAdditionalWorkerNodePoolsSchema(additionalMachineTypesDisplay, additionalMachineTypes, rejectUnsupportedParameters),
		},
		Region: &Type{
			Type:            "string",
			Enum:            ToInterfaceSlice(regions),
			EnumDisplayName: regionsDisplay,
			MinLength:       1,
		},
		Networking:           NewNetworkingSchema(rejectUnsupportedParameters, providerSpec, cloudProvider, dualStackDocsURL),
		Modules:              NewModulesSchema(rejectUnsupportedParameters),
		ColocateControlPlane: ColocateControlPlaneProperty(),
	}

	if update {
		properties.AutoScalerMax.Default = nil
		properties.AutoScalerMin.Default = nil
	}

	return properties
}

func NewNetworkingSchema(rejectUnsupportedParameters bool, providerSpec *configuration.ProviderSpec, cloudProvider pkg.CloudProvider, dualStackDocsURL string) *NetworkingType {
	seedCIDRs := strings.Join(networking.GardenerSeedCIDRs, ", ")
	networkingProperties := NetworkingProperties{
		Services: Type{Type: "string", Title: "CIDR range for Services", Description: fmt.Sprintf("CIDR range for Services, must not overlap with the following CIDRs: %s", seedCIDRs),
			Default: networking.DefaultServicesCIDR},
		Pods: Type{Type: "string", Title: "CIDR range for Pods", Description: fmt.Sprintf("CIDR range for Pods, must not overlap with the following CIDRs: %s", seedCIDRs),
			Default: networking.DefaultPodsCIDR},
		Nodes: Type{Type: "string", Title: "CIDR range for Nodes", Description: fmt.Sprintf("CIDR range for Nodes, must not overlap with the following CIDRs: %s", seedCIDRs),
			Default: networking.DefaultNodesCIDR},
	}

	if providerSpec != nil && providerSpec.IsDualStackSupported(cloudProvider) {
		description := "Enable dual-stack networking (IPv4 and IPv6). The Kyma Istio module does not support dual-stack mode."
		if dualStackDocsURL != "" {
			description += fmt.Sprintf(" For more information, see <a href=%s>documentation</a>.", dualStackDocsURL)
		}
		networkingProperties.DualStack = &Type{Type: "boolean", Title: "Enable dual stack (Istio module doesn't support dual stack)", Description: description}
	}

	networkingType := &NetworkingType{
		Type:       Type{Type: "object", Description: "Networking configuration. These values are immutable and cannot be updated later. All provided CIDR ranges must not overlap one another."},
		Properties: networkingProperties,
		Required:   []string{"nodes"},
	}
	if rejectUnsupportedParameters {
		networkingType.Type.AdditionalProperties = false
	}
	return networkingType
}

func NewSchema(properties interface{}, required []string, rejectUnsupportedParameters bool) *RootSchema {
	rootSchema := &RootSchema{
		Schema: "http://json-schema.org/draft-04/schema#",
		Type: Type{
			Type: "object",
		},
		Properties:        properties,
		ShowFormView:      true,
		Required:          required,
		LoadCurrentConfig: true,
	}
	if rejectUnsupportedParameters {
		rootSchema.Type.AdditionalProperties = false
	}
	return rootSchema
}

func unmarshalOrPanic(from, to interface{}) interface{} {
	if from != nil {
		marshaled := Marshal(from)
		err := json.Unmarshal(marshaled, to)
		if err != nil {
			panic(err)
		}
	}
	return to
}

func DefaultControlsOrder() []string {
	return []string{"name", "kubeconfig", "shootName", "shootDomain", "region", "colocateControlPlane", "machineType", "autoScalerMin", "autoScalerMax", "zonesCount", "additionalWorkerNodePools", "modules", "networking", "oidc", "administrators", "ingressFiltering"}
}

func ToInterfaceSlice(input []string) []interface{} {
	interfaces := make([]interface{}, len(input))
	for i, item := range input {
		interfaces[i] = item
	}
	return interfaces
}

func AdministratorsProperty() *Type {
	return &Type{
		Type:        "array",
		Title:       "Administrators",
		Description: "Specifies the list of runtime administrators.",
		Items: &Type{
			Type: "string",
		},
	}
}

func NewAdditionalWorkerNodePoolsSchema(machineTypesDisplay map[string]string, machineTypes []string, rejectUnsupportedParameters bool) *AdditionalWorkerNodePoolsType {
	additionalWorkerNodePoolsType := &AdditionalWorkerNodePoolsType{
		Type: Type{
			Type:        "array",
			UniqueItems: true,
			Description: "Specifies the list of additional worker node pools."},
		Items: AdditionalWorkerNodePoolsItems{
			ControlsOrder: []string{"name", "machineType", "haZones", "autoScalerMin", "autoScalerMax"},
			Required:      []string{"name", "machineType", "haZones", "autoScalerMin", "autoScalerMax"},
			Type: Type{
				Type: "object",
			},
			Properties: AdditionalWorkerNodePoolsItemsProperties{
				Name: Type{
					Type:        "string",
					MinLength:   1,
					MaxLength:   15,
					Pattern:     "^(?!cpu-worker-0$)[a-z0-9]([-a-z0-9]*[a-z0-9])?$",
					Description: "Specifies the unique name of the additional worker node pool. The name must consist of lowercase alphanumeric characters or '-', must start and end with an alphanumeric character, and can be a maximum of 15 characters in length. Do not use the name “cpu-worker-0” because it's reserved for the Kyma worker node pool.",
				},
				MachineType: Type{
					Type:            "string",
					MinLength:       1,
					Enum:            ToInterfaceSlice(machineTypes),
					EnumDisplayName: machineTypesDisplay,
					Description:     "Specifies the type of the virtual machine. The machine type marked with “*” has limited availability and generates high cost. This setting is permanent, and you cannot change it later. To use a different machine type, you must create a new worker node pool, migrate workloads to it, and decommission the old one.",
				},
				HAZones: &Type{
					Type:        "boolean",
					Title:       "HA zones",
					Default:     true,
					Description: "Specifies whether high availability (HA) zones are supported. This setting is permanent and cannot be changed later. If HA is disabled, all resources are placed in a single, randomly selected zone. Disabled HA allows setting autoScalerMin to 0 and autoScalerMax to 1, which helps reduce costs. It is not recommended for production environments. When enabled, resources are distributed across three zones to enhance fault tolerance. Enabled HA requires setting autoScalerMin to the minimal value 3.",
				},
				AutoScalerMin: AutoscalerType{
					Type:        "integer",
					Minimum:     nonHAAutoscalerMinMinimumValue,
					Maximum:     autoscalerMaximumValue,
					Default:     pkg.HAAutoscalerMinimumValue,
					Description: "Specifies the minimum number of virtual machines to create.",
				},
				AutoScalerMax: AutoscalerType{
					Type:        "integer",
					Minimum:     nonHAAutoscalerMaxMinimumValue,
					Maximum:     autoscalerMaximumValue,
					Default:     autoscalerMaxDefaultValue,
					Description: "Specifies the maximum number of virtual machines to create.",
				},
			},
		},
	}
	if rejectUnsupportedParameters {
		additionalWorkerNodePoolsType.Items.Type.AdditionalProperties = false
	}
	return additionalWorkerNodePoolsType
}

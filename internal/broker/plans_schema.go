package broker

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kyma-project/kyma-environment-broker/internal/networking"
)

type RootSchema struct {
	Schema string `json:"$schema"`
	Type
	Properties interface{} `json:"properties"`
	Required   []string    `json:"required"`

	// Specified to true enables form view on website
	ShowFormView bool `json:"_show_form_view"`
	// Specifies in what order properties will be displayed on the form
	ControlsOrder []string `json:"_controlsOrder"`
	// Specified to true loads current instance configuration into the update instance schema
	LoadCurrentConfig bool `json:"_load_current_config,omitempty"`
}

type ProvisioningProperties struct {
	UpdateProperties

	Name                   NameType        `json:"name"`
	ShootName              *Type           `json:"shootName,omitempty"`
	ShootDomain            *Type           `json:"shootDomain,omitempty"`
	Region                 *Type           `json:"region,omitempty"`
	Networking             *NetworkingType `json:"networking,omitempty"`
	Modules                *Modules        `json:"modules,omitempty"`
	ShootAndSeedSameRegion *Type           `json:"shootAndSeedSameRegion,omitempty"`
}

type UpdateProperties struct {
	Kubeconfig                *Type                          `json:"kubeconfig,omitempty"`
	AutoScalerMin             *Type                          `json:"autoScalerMin,omitempty"`
	AutoScalerMax             *Type                          `json:"autoScalerMax,omitempty"`
	OIDC                      *OIDCType                      `json:"oidc,omitempty"`
	Administrators            *Type                          `json:"administrators,omitempty"`
	MachineType               *Type                          `json:"machineType,omitempty"`
	AdditionalWorkerNodePools *AdditionalWorkerNodePoolsType `json:"additionalWorkerNodePools,omitempty"`
}

func (up *UpdateProperties) IncludeAdditional() {
	up.OIDC = NewOIDCSchema()
	up.Administrators = AdministratorsProperty()
}

type NetworkingProperties struct {
	Nodes    Type `json:"nodes"`
	Services Type `json:"services"`
	Pods     Type `json:"pods"`
}

type NetworkingType struct {
	Type
	Properties NetworkingProperties `json:"properties"`
	Required   []string             `json:"required"`
}

type OIDCProperties struct {
	ClientID       Type `json:"clientID"`
	GroupsClaim    Type `json:"groupsClaim"`
	IssuerURL      Type `json:"issuerURL"`
	SigningAlgs    Type `json:"signingAlgs"`
	UsernameClaim  Type `json:"usernameClaim"`
	UsernamePrefix Type `json:"usernamePrefix"`
}

type OIDCType struct {
	Type
	Properties OIDCProperties `json:"properties"`
	Required   []string       `json:"required"`
}

type Type struct {
	Type        string `json:"type"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Minimum     int    `json:"minimum,omitempty"`
	Maximum     int    `json:"maximum,omitempty"`
	MinLength   int    `json:"minLength,omitempty"`
	MaxLength   int    `json:"maxLength,omitempty"`

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

type NameType struct {
	Type
	BTPdefaultTemplate BTPdefaultTemplate `json:"_BTPdefaultTemplate,omitempty"`
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
	Name          Type  `json:"name,omitempty"`
	MachineType   Type  `json:"machineType,omitempty"`
	HAZones       *Type `json:"haZones,omitempty"`
	AutoScalerMin Type  `json:"autoScalerMin,omitempty"`
	AutoScalerMax Type  `json:"autoScalerMax,omitempty"`
}

func NewModulesSchema() *Modules {
	return &Modules{
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
}

func NameProperty() NameType {
	return NameType{
		Type: Type{
			Type:  "string",
			Title: "Cluster Name",
			// Allows for all alphanumeric characters and '-'
			Pattern:   "^[a-zA-Z0-9-]*$",
			MinLength: 1,
		},
		BTPdefaultTemplate: BTPdefaultTemplate{
			Elements: []string{"saSubdomain"},
		},
	}
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

func ShootAndSeedSameRegionProperty() *Type {
	return &Type{
		Type:        "boolean",
		Title:       "Enforce Same Location for Seed and Shoot",
		Default:     false,
		Description: "If set to true a Gardener seed will be placed in the same region as the selected region from the Region field. Provisioning process will fail if no seed is availabie in the region.",
	}
}

// NewProvisioningProperties creates a new properties for different plans
// Note that the order of properties will be the same in the form on the website
func NewProvisioningProperties(machineTypesDisplay, additionalMachineTypesDisplay, regionsDisplay map[string]string, machineTypes, additionalMachineTypes, regions []string, update bool) ProvisioningProperties {

	properties := ProvisioningProperties{
		UpdateProperties: UpdateProperties{
			AutoScalerMin: &Type{
				Type:        "integer",
				Minimum:     3,
				Default:     3,
				Description: "Specifies the minimum number of virtual machines to create",
			},
			AutoScalerMax: &Type{
				Type:        "integer",
				Minimum:     3,
				Maximum:     300,
				Default:     20,
				Description: "Specifies the maximum number of virtual machines to create",
			},
			MachineType: &Type{
				Type:            "string",
				Enum:            ToInterfaceSlice(machineTypes),
				EnumDisplayName: machineTypesDisplay,
				Description:     "Specifies the type of the virtual machine.",
			},
			AdditionalWorkerNodePools: NewAdditionalWorkerNodePoolsSchema(additionalMachineTypesDisplay, additionalMachineTypes),
		},
		Name: NameProperty(),
		Region: &Type{
			Type:            "string",
			Enum:            ToInterfaceSlice(regions),
			EnumDisplayName: regionsDisplay,
			MinLength:       1,
		},
		Networking: NewNetworkingSchema(),
		Modules:    NewModulesSchema(),
	}

	if update {
		properties.AutoScalerMax.Default = nil
		properties.AutoScalerMin.Default = nil
	}

	return properties
}

func NewNetworkingSchema() *NetworkingType {
	seedCIDRs := strings.Join(networking.GardenerSeedCIDRs, ", ")
	return &NetworkingType{
		Type: Type{Type: "object", Description: "Networking configuration. These values are immutable and cannot be updated later. All provided CIDR ranges must not overlap one another."},
		Properties: NetworkingProperties{
			Services: Type{Type: "string", Title: "CIDR range for Services", Description: fmt.Sprintf("CIDR range for Services, must not overlap with the following CIDRs: %s", seedCIDRs),
				Default: networking.DefaultServicesCIDR},
			Pods: Type{Type: "string", Title: "CIDR range for Pods", Description: fmt.Sprintf("CIDR range for Pods, must not overlap with the following CIDRs: %s", seedCIDRs),
				Default: networking.DefaultPodsCIDR},
			Nodes: Type{Type: "string", Title: "CIDR range for Nodes", Description: fmt.Sprintf("CIDR range for Nodes, must not overlap with the following CIDRs: %s", seedCIDRs),
				Default: networking.DefaultNodesCIDR},
		},
		Required: []string{"nodes"},
	}
}

func NewOIDCSchema() *OIDCType {
	return &OIDCType{
		Type: Type{Type: "object", Description: "OIDC configuration"},
		Properties: OIDCProperties{
			ClientID:       Type{Type: "string", Description: "The client ID for the OpenID Connect client."},
			IssuerURL:      Type{Type: "string", Description: "The URL of the OpenID issuer, only HTTPS scheme will be accepted."},
			GroupsClaim:    Type{Type: "string", Description: "If provided, the name of a custom OpenID Connect claim for specifying user groups."},
			UsernameClaim:  Type{Type: "string", Description: "The OpenID claim to use as the user name."},
			UsernamePrefix: Type{Type: "string", Description: "If provided, all usernames will be prefixed with this value. If not provided, username claims other than 'email' are prefixed by the issuer URL to avoid clashes. To skip any prefixing, provide the value '-' (dash character without additional characters)."},
			SigningAlgs: Type{
				Type: "array",
				Items: &Type{
					Type: "string",
				},
				Description: "Comma separated list of allowed JOSE asymmetric signing algorithms, for example, RS256, ES256",
			},
		},
		Required: []string{"clientID", "issuerURL"},
	}
}

func NewSchema(properties interface{}, update bool, required []string) *RootSchema {
	schema := &RootSchema{
		Schema: "http://json-schema.org/draft-04/schema#",
		Type: Type{
			Type: "object",
		},
		Properties:        properties,
		ShowFormView:      true,
		Required:          required,
		LoadCurrentConfig: true,
	}

	if update {
		schema.Required = []string{}
	}

	return schema
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
	return []string{"name", "kubeconfig", "shootName", "shootDomain", "region", "shootAndSeedSameRegion", "machineType", "autoScalerMin", "autoScalerMax", "zonesCount", "additionalWorkerNodePools", "modules", "networking", "oidc", "administrators"}
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
		Description: "Specifies the list of runtime administrators",
		Items: &Type{
			Type: "string",
		},
	}
}

func NewAdditionalWorkerNodePoolsSchema(machineTypesDisplay map[string]string, machineTypes []string) *AdditionalWorkerNodePoolsType {
	return &AdditionalWorkerNodePoolsType{
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
					Type:      "string",
					MinLength: 1,
					MaxLength: 15,
					// Allows for all alphanumeric characters and '-'
					Pattern:     "^[a-z0-9]([-a-z0-9]*[a-z0-9])?$",
					Description: "Specifies the unique name of the additional worker node pool. The name must consist of lowercase alphanumeric characters or '-', must start and end with an alphanumeric character, and can be a maximum of 15 characters in length.",
				},
				MachineType: Type{
					Type:            "string",
					MinLength:       1,
					Enum:            ToInterfaceSlice(machineTypes),
					EnumDisplayName: machineTypesDisplay,
					Description:     "Specifies the type of the virtual machine.",
				},
				HAZones: &Type{
					Type:        "boolean",
					Title:       "HA zones",
					Default:     true,
					Description: "Specifies whether high availability (HA) zones are supported. This setting is permanent and cannot be changed later. If HA is disabled, all resources are placed in a single, randomly selected zone. Disabled HA allows setting both autoScalerMin and autoScalerMax to 1, which helps reduce costs. It is not recommended for production environments. When enabled, resources are distributed across three zones to enhance fault tolerance. Enabled HA requires setting autoScalerMin to the minimal value 3.",
				},
				AutoScalerMin: Type{
					Type:        "integer",
					Minimum:     1,
					Default:     3,
					Description: "Specifies the minimum number of virtual machines to create.",
				},
				AutoScalerMax: Type{
					Type:        "integer",
					Minimum:     1,
					Maximum:     300,
					Default:     20,
					Description: "Specifies the maximum number of virtual machines to create.",
				},
			},
		},
	}
}

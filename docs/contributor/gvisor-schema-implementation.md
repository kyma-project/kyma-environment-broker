# Feature: gVisor Schema Property — Implementation Notes

Branch: `feature/add-gvisor-for-worker-pools`

---

## Goal

Add the following JSON schema object to **every service plan** exposed by the broker's OSB catalog endpoint (`GET /v2/catalog`), in both the **create** and **update** schemas:

```json
{
  "gvisor": {
    "type": "object",
    "title": "gVisor container runtime sandbox",
    "description": "Configures the gVisor container runtime sandbox for a worker pool",
    "required": ["enabled"],
    "properties": {
      "enabled": {
        "type": "boolean",
        "title": "Enable gVisor container runtime sandbox",
        "default": false
      }
    }
  }
}
```

---

## How the JSON Schema Is Built

### Data flow: config files → Go structs → `map[string]interface{}`

```
resources/keb/templates/app-config.yaml   (Helm ConfigMap)
  ├── providersConfig.yaml  →  configuration.ProviderSpec
  │     regions, zone lists, machine display names, dualStack flag
  └── plansConfig.yaml      →  configuration.PlanSpecifications
        regions per platform-region, regularMachines, additionalMachines,
        upgradableToPlans, volumeSizeGb
```

Both files are loaded at startup in `cmd/broker/main.go` and injected into `SchemaService`.

### `SchemaService` (`internal/broker/schemas.go`)

`NewSchemaService(providerSpec, planSpec, defaultOIDC, cfg, ingressFilteringPlans, channelResolver)`

Key method for the OSB catalog:

```go
func (s *SchemaService) Plans(plansConfig PlansConfig, platformRegion string, cp CloudProvider) map[string]domain.ServicePlan
```

For each plan it calls a plan-specific method which falls into one of three code paths:

| Path | Plans | Method |
|------|-------|--------|
| **`planSchemas()`** | aws, gcp, azure, sap-converged-cloud, alicloud, preview, build-runtime-* | `AWSSchemas`, `GCPSchemas`, `AzureSchemas`, … |
| **`AzureLiteSchema()`** | azure_lite | custom — mutates `ProvisioningProperties` after construction |
| **`FreeSchema()` / `TrialSchema()`** | free, trial | custom — builds `ProvisioningProperties` manually |

All three paths end by calling:

```go
createSchemaWithProperties(properties ProvisioningProperties, defaultOIDC, update bool, required []string, flags)
  → createSchemaWith(properties / properties.UpdateProperties, required, rejectUnsupportedParameters)
    → unmarshalSchema(NewSchema(properties, required, rejectUnsupportedParameters))
      → json.Marshal struct → json.Unmarshal into map[string]interface{}
      → filter _controlsOrder to only keys present in properties
```

**Create schema** serialises the full `ProvisioningProperties` struct.  
**Update schema** serialises only the embedded `UpdateProperties` sub-struct (no `region`, `networking`, `modules`, `colocateControlPlane`).

### Key structs (`internal/broker/plans_schema.go`)

```
RootSchema
└── properties: ProvisioningProperties   (create)
              | UpdateProperties          (update)

ProvisioningProperties embeds UpdateProperties, plus:
  ShootName, ShootDomain, Region, Networking, Modules, ColocateControlPlane

UpdateProperties:
  Name, Kubeconfig, AutoScalerMin, AutoScalerMax, OIDC,
  Administrators, MachineType, AdditionalWorkerNodePools,
  IngressFiltering, Gvisor   ← added in this feature
```

### `createSchemaWithProperties` — where unconditional properties are wired in

```go
// internal/broker/plans.go
func createSchemaWithProperties(properties ProvisioningProperties, ...) *map[string]interface{} {
    properties.OIDC = NewMultipleOIDCSchema(...)
    properties.Administrators = AdministratorsProperty()
    properties.Gvisor = GvisorProperty()          // ← added in this feature
    if flags.ingressFilteringEnabled {
        properties.IngressFiltering = IngressFilteringProperty()
    }
    ...
}
```

### `DefaultControlsOrder` — controls BTP UI form field order

```go
// internal/broker/plans_schema.go  line 764
func DefaultControlsOrder() []string {
    return []string{
        "name", "kubeconfig", "shootName", "shootDomain", "region",
        "colocateControlPlane", "machineType", "autoScalerMin", "autoScalerMax",
        "zonesCount", "additionalWorkerNodePools", "modules", "networking",
        "oidc", "administrators", "ingressFiltering",
        // "gvisor" is NOT yet here — see TODO below
    }
}
```

After `unmarshalSchema()`, any key in `_controlsOrder` that is not present in `properties` is **silently removed** by the `filter()` function. So adding `"gvisor"` to `DefaultControlsOrder()` is required for it to appear in the rendered `_controlsOrder` field.

---

## What Has Been Done

### 1. New Go types (`internal/broker/plans_schema.go`)

```go
// Added to UpdateProperties struct (line 56):
Gvisor *GvisorType `json:"gvisor,omitempty"`

// New types (lines 59–67):
type GvisorProperties struct {
    Enabled Type `json:"enabled"`
}

type GvisorType struct {
    Type
    Required   []string         `json:"required"`
    Properties GvisorProperties `json:"properties"`
}

// New constructor (lines 885–901):
func GvisorProperty() *GvisorType {
    return &GvisorType{
        Type: Type{
            Type:        "object",
            Title:       "gVisor container runtime sandbox",
            Description: "Configures the gVisor container runtime sandbox for a worker pool",
        },
        Required: []string{"enabled"},
        Properties: GvisorProperties{
            Enabled: Type{
                Type:    "boolean",
                Title:   "Enable gVisor container runtime sandbox",
                Default: false,
            },
        },
    }
}
```

### 2. Wired into all plan schemas (`internal/broker/plans.go`, line 158)

```go
properties.Gvisor = GvisorProperty()
```

This single line covers all plans because every schema code path passes through `createSchemaWithProperties`.

### 3. Unit tests (`internal/broker/plans_test.go`)

**`TestSchemaService_GvisorPropertyPresentInAllPlans`** — table-driven test with 20 sub-cases covering every distinct code path:

| Sub-cases | Code path tested |
|-----------|-----------------|
| aws, gcp, azure, sap-converged-cloud, alicloud, preview (create + update) | `planSchemas()` |
| azure-lite (create + update) | `AzureLiteSchema()` |
| free-aws, free-azure (create + update) | `FreeSchema()` |
| trial (create + update) | `TrialSchema()` |

Helper function `gvisorProperty()` returns the expected `map[string]interface{}` for assertion.

Platform region constants defined to avoid repeated string literals:

```go
const (
    platformRegionUS  = "cf-us11"   // AWS, GCP, Preview
    platformRegionUS2 = "cf-us21"   // Azure, Azure Lite, Free
    platformRegionEU  = "cf-eu20"   // SAP Converged Cloud
    platformRegionEU2 = "cf-eu40"   // Alicloud
)
```

**Status: all 20 sub-tests pass. ✅**

---

## What Remains To Be Done

### TODO 1 — Add `"gvisor"` to `DefaultControlsOrder()`

**File:** `internal/broker/plans_schema.go`, function `DefaultControlsOrder()` (line 764).

Decide where in the order `gvisor` should appear (e.g. at the end, after `ingressFiltering`):

```go
func DefaultControlsOrder() []string {
    return []string{
        "name", "kubeconfig", "shootName", "shootDomain", "region",
        "colocateControlPlane", "machineType", "autoScalerMin", "autoScalerMax",
        "zonesCount", "additionalWorkerNodePools", "modules", "networking",
        "oidc", "administrators", "ingressFiltering", "gvisor",
    }
}
```

### TODO 2 — Add `_controlsOrder` unit test

Re-add `TestSchemaService_GvisorInControlsOrder` (was written and then removed for the small-steps approach). Same case table as the property-presence test. Assert:

```go
order, ok := (*schema)["_controlsOrder"].([]interface{})
require.True(t, ok, "schema has no '_controlsOrder' key")
assert.Contains(t, order, "gvisor")
```

### TODO 3 — Update golden files

All existing golden-file snapshot tests (`TestSchemaService_Aws`, `TestSchemaService_Azure`, etc. in `plans_test.go`) will now fail because `gvisor` is present in the generated schema but absent from the stored `.json` files in `testdata/`.

For each affected file:
1. Run `go test ./internal/broker/... -run TestSchemaService_<Plan> -v` to get the actual output
2. Copy the "Actual" JSON from the test failure diff into the corresponding `testdata/<provider>/*.json` file

Affected directories:
- `internal/broker/testdata/aws/`
- `internal/broker/testdata/azure/`
- `internal/broker/testdata/gcp/`
- `internal/broker/testdata/sap-converged-cloud/`
- `internal/broker/testdata/alicloud/`

### TODO 4 — Process the `gvisor` parameter in provisioning/update steps

The schema addition makes the parameter accepted by the OSB API, but the broker also needs to **read** the value from the provisioning/update request and **pass it** to the RuntimeResource CR that Kyma Infrastructure Manager creates.

Relevant files to look at:
- `internal/process/provisioning/create_runtime_resource_step.go` — creates the KIM `RuntimeResource` CR; worker pool configuration is assembled here
- `internal/process/update/` — similar logic for update operations
- `internal/workers/` — `WorkersProvider` and worker node pool config

The `gvisor.enabled` value from the request input params needs to flow into the `RuntimeResource` spec for each worker pool. Look at how existing worker pool fields (e.g. `machineType`, `autoScalerMin`) are transferred from `internal.ProvisioningParameters` into the KIM CR.

---

## File Map

| File | Role |
|------|------|
| `internal/broker/plans_schema.go` | All schema Go types + property constructors + `DefaultControlsOrder` |
| `internal/broker/plans.go` | Plan IDs/names, `createSchemaWithProperties`, `createSchemaWith`, `unmarshalSchema` |
| `internal/broker/schemas.go` | `SchemaService` — per-plan schema methods, `planSchemas` shared path |
| `internal/broker/services.go` | OSB `GET /v2/catalog` handler — calls `schemaService.Plans()` |
| `internal/broker/plans_test.go` | All schema unit tests + `createSchemaService` fixture + platform region constants |
| `internal/broker/testdata/plans.yaml` | Test plan config (machines, regions per plan) |
| `internal/broker/testdata/providers.yaml` | Test provider config (region/machine display names) |
| `internal/broker/testdata/<provider>/` | Golden JSON files for snapshot tests |
| `internal/provider/configuration/plan.go` | `PlanSpecifications` — parses `plansConfig.yaml` |
| `internal/provider/configuration/provider.go` | `ProviderSpec` — parses `providersConfig.yaml` |
| `resources/keb/templates/app-config.yaml` | Helm ConfigMap — mounts both YAML configs into the pod |
| `resources/keb/values.yaml` | Helm values — `plansConfiguration` and `providersConfiguration` keys |

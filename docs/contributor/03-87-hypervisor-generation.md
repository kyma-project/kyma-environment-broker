<!--{"metadata":{"publish":false}}-->

# Hypervisor Generation Support for Azure

## Overview

Some Azure machine types require a Generation 2 (Gen2) hypervisor-compatible machine image version. For example, `Standard_D2s_v6` cannot boot with a Gen1 image and requires the `-gen2` image variant. Kyma Environment Broker (KEB) automatically detects the supported hypervisor generation for each Azure machine type and appends the appropriate suffix (for example, `-gen2`) to the machine image version when creating or updating a Runtime CR.

> ### Note:
> This feature is Azure-only. For other infrastructure providers, the machine image version is used as-is.

## How It Works

### Detection

During provisioning or update, the [`Discover_Available_Zones_CredentialsBinding`](../../internal/process/steps/discover_available_zones_cb.go) step queries the Azure ResourceSKUs API for the instance's subscription and region. For each machine type involved in the operation, it reads the `HyperVGenerations` capability from the SKU response. The possible values are:

- `V1` — Gen1 only, no suffix appended
- `V2` — Gen2 only, suffix `-gen2` appended
- `V1,V2` — both generations supported, suffix `-gen2` appended (newest generation preferred)
- missing — no suffix appended

The detected suffixes are stored in `operation.MachineImageVersionSuffixes`.

### Application

The suffix is concatenated with the base `machineImageVersion` config value when writing the Runtime CR:

- **Provisioning** ([`Create_Runtime_Resource`](../../internal/process/provisioning/create_runtime_resource_step.go)): applied to the Kyma worker and all additional worker node pools.
- **Update** ([`Update_Runtime_Resource`](../../internal/process/update/update_runtime_step.go)):
  - Kyma worker: suffix applied when the machine type changes; otherwise the existing image version is kept.
  - Additional worker node pools: suffix applied when a pool is new or its machine type changes; existing unchanged pools keep their existing image version.

Example: if `machineImageVersion` is `2150.3.0` and the machine type supports Gen2, the Runtime CR receives `2150.3.0-gen2`.

### Caching

The Azure ResourceSKUs API call is shared between zone discovery and hypervisor generation detection — a single API call populates both caches. The cache is per-step-execution (not global), scoped to the subscription and region of the instance.

## Configuration

| Environment Variable                                          | Helm Value                                           | Default | Description                                                         |
|---------------------------------------------------------------|------------------------------------------------------|---------|---------------------------------------------------------------------|
| `APP_INFRASTRUCTURE_MANAGER_USE_MACHINE_IMAGE_VERSION_SUFFIX` | `infrastructureManager.useMachineImageVersionSuffix` | `false` | Enables automatic machine image version suffix detection for Azure. |

## Upgrading `machineImageVersion`

When upgrading the `machineImageVersion` (for example, from `2150.3.0` to a newer release), verify whether the new version also has a corresponding `-gen2` variant available in Azure. KEB selects the suffix based on what the Azure API reports for the SKU — if a Gen2 image variant is not published for the new version, instances using Gen2-capable machine types will fail to provision or update.

If an existing worker already uses a `-gen2` image version, that suffix must be preserved after the upgrade. If a worker does not use `-gen2`, the suffix must not be introduced during the upgrade.

## Upgrading Machine Type Generation

When the `machinesVersions` mapping is updated to point a version-agnostic name to a newer concrete machine type (for example, `Standard_D{size}s` remapped to `Standard_D{size}s_v6`), the resolved machine type may support a different hypervisor generation. The suffix must reflect the generation supported by the new resolved machine type:

| Previous generation | New generation | Suffix |
|---|---|---|
| Gen1 | Gen1 | no suffix |
| Gen1 | Gen2 | `-gen2` must be added |
| Gen2 | Gen2 | `-gen2` must be preserved |

> ### Note:
> KEB does not automatically reconcile existing worker pools when `machinesVersions` changes. The suffix is only re-evaluated when a provisioning or update operation is triggered for the instance.

## Introducing New Machine Types

When new machine types are added to the provider configuration, no additional action is required — KEB queries the Azure API dynamically for each machine type present in the provisioning or update request.

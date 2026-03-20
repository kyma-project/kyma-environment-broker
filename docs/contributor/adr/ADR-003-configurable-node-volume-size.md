# Configurable Node Volume Size

## Status

Proposed

## Context

If the disk (volume) attached to Kubernetes worker nodes fills up, it can render the node unusable and cause workload disruptions.

Users running large machine types with many pods can encounter disk-full conditions because the current default volume size (80 GiB) is insufficient for high-density nodes. Today, `volumeSizeGb` is an internal KEB setting per plan and users cannot configure it.

### Current State

- `volumeSizeGb` is defined per plan in the KEB plans configuration.
- The value is not exposed to users, neither for the main worker pool nor for additional worker pools.
- `sap-converged-cloud` does not set any volume configuration.

### Why Not a Dynamic Default Based on Machine Size

A formula-based approach was discussed (e.g., `volume_base + max(vCPUs/2, memory_GiB/8) * volume_factor`), but was ruled out because the BTP cockpit cannot display dynamic default values. A static default must be shown to the user.

## Decision

Expose `volumeSizeGb` as an optional, user-configurable parameter per worker pool and for the main worker. The value cannot be set below the plan default. When not provided, the plan default applies unchanged.

### 1. New Parameter: `volumeSizeGb`

A new optional integer parameter `volumeSizeGb` is added to:
- The **main provisioning parameters** (system/main worker pool)
- Each entry in the **`additionalWorkerNodePools`** array

When omitted, the existing KEB plan default applies (backward compatible). When provided, it must be >= the plan default.

**BTP cockpit - main worker pool:**

![volumeSizeGb in main worker pool](../../assets/adr-003-volume-size-main-worker-pool.png)

**BTP cockpit - additional worker node pool:**

![volumeSizeGb in additional worker node pool](../../assets/adr-003-volume-size-additional-worker-pool.png)

**Provisioning request example:**
```json
{
  "parameters": {
    "name": "my-cluster",
    "machineType": "m6i.4xlarge",
    "volumeSizeGb": 150,
    "additionalWorkerNodePools": [
      {
        "name": "gpu-pool",
        "machineType": "g4dn.8xlarge",
        "volumeSizeGb": 200,
        "autoScalerMin": 1,
        "autoScalerMax": 3
      }
    ]
  }
}
```

### 2. Validation Rules

| Rule | Details |
|------|---------|
| Minimum value | Must be >= plan default. |
| Maximum value | Optional upper bound to prevent accidental excessive cost |
| Omitted value | Defaults to the plan's configured `volumeSizeGb` (current behavior preserved). |
| Update operations | The value can be changed on update. **Should we allow to decrease it? Does Gardener allow it?** |

### 3. Schema Changes

The JSON schema must include `volumeSizeGb` as an optional integer property:
- In the top-level provisioning/update parameters (for the main worker pool)
- In each item of `additionalWorkerNodePools`

The schema declares `minimum` and `maximum` constraints as static values to communicate the allowed range to consumers via the BTP cockpit.

### 4. Defaulting Behavior

- If `volumeSizeGb` is **not** included in the request, KEB defaults to the plan-level configuration value.
- The default value is displayed to the user in the BTP cockpit so they know what they get if they don't specify it.
- When operators change the KEB default for a plan, they must also update ERS to reflect the new default.

### 5. Billing

- The plan default volume size is included in the base machine price with no additional charge.
- Any volume size **above** the plan default is charged per GB difference, per node, per month.
- The per-GB price should align with the provider's persistent volume pricing.
- The price calculator must be updated to include a `volumeSizeGb` input field, showing the additional cost when the value exceeds the default.

## Consequences

- Users gain the ability to prevent disk-full conditions by requesting more volume per worker pool.
- Users who don't need extra disk are unaffected, no cost increase, no behavioral change.
- Billing becomes more granular: users pay only for storage above the included default.
- `sap-converged-cloud` remains unchanged; the parameter is not available for that plan.

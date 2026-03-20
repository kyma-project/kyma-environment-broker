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

## Approach 1: Static `volumeSizeGb` per Worker Pool

Expose `volumeSizeGb` as an optional, user-configurable integer parameter per worker pool and for the main worker. The value cannot be set below the plan default. When not provided, the plan default applies unchanged.

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

The schema declares `minimum` and `maximum` constraints as static values. The default and minimum are shown in the BTP cockpit. Users pay for every GB above the plan default.

### Pros

- Simple to implement.
- Clear billing: users pay extra only for GB above the plan default.
- No machine-type metadata required.
- User has more control over the size

### Cons

- TBD

## Approach 2: Dynamic Volume Based on Machine Type + `additionalVolumeGb`

KEB computes a volume size automatically based on the selected machine type (using a formula or lookup table, e.g., `volume_base + max(vCPUs/2, memory_GiB/8) * volume_factor`). The computed size is included in the base machine price at no extra cost. Users can additionally request extra GB on top via an optional `additionalVolumeGb` parameter, which is billed separately per GB above the computed base.

**Provisioning request example:**
```json
{
  "parameters": {
    "name": "my-cluster",
    "machineType": "m6i.4xlarge",
    "additionalVolumeGb": 50,
    "additionalWorkerNodePools": [
      {
        "name": "gpu-pool",
        "machineType": "g4dn.8xlarge",
        "additionalVolumeGb": 100,
        "autoScalerMin": 1,
        "autoScalerMax": 3
      }
    ]
  }
}
```

To make the computed default transparent, each machine type option displayed in the BTP cockpit (the `machineType` enum) would need to include the resulting volume size alongside vCPU and memory information, e.g.:

```
m6i.4xlarge (16 vCPU, 64 GiB RAM, 148 GiB disk)
```

### Pros

- Large machines automatically get a larger disk without any user action.
- Billing is clearly scoped to the additional portion only.

### Cons

- More complex to implement.
- Different formula parameters needed per OS image? (GardenLinux vs. Ubuntu Pro).


## Decision

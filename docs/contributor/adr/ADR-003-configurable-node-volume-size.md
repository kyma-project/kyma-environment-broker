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

There are two variants for what we could expose to the user:

**Variant A: Expose `volumeSizeGb` directly** - the user sets the total volume size. The schema shows the default and minimum (equal to the plan default). Users see and manage the full size. When the parameter is not provided in the payload, the plan default is applied for backwards compatibility.

**Variant B: Expose only `additionalVolumeGb`** - the user sets only the extra GB on top of the plan default. The input defaults to 0. The plan default base is transparent to the user and always included for free. Tthe input directly represents what the user pays for.

The screenshots below show Variant A. For Variant B the UI looks almost the same, but the default value would be 0 and the label would be `Additional Volume Gb`.

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

The schema declares `minimum` and `maximum` constraints as static values. The default and minimum are shown in the BTP cockpit.

### Billing

- The plan default volume size is included in the base machine price with no additional charge.
- Any `volumeSizeGb` value above the plan default is charged.
- The volume size is available as a status attribute on the Kubernetes node API, so the actual disk size per node can be detected at runtime for billing purposes.
- The price calculator must be updated to include a `volumeSizeGb` input field showing the additional cost when the value exceeds the default.

### Pros

- Simple to implement and maintain.
- Users pay extra only for GB above the plan default.
- No calculations of the volume size per machine are needed.
- KEB operators need to refresh ERS only once when this feature is rolled out or when the default value is changed.

### Cons

- TBD

## Approach 2: Dynamic Volume Based on Machine Type + `additionalVolumeGb`

KEB computes a volume size automatically based on the selected machine type. The computed size is included in the base machine price at no extra cost. Users can additionally request extra GB on top via an optional `additionalVolumeGb` parameter, which is billed separately per GB above the computed base.

The dynamic volume size can be obtained using one of two sub-options for the calculation:

**Sub-option A: Configurable mapping table** — maps machine size ranges (e.g., by vCPU count) to fixed volume sizes. Easier to reason about but requires maintaining the table as new machine types are added.

**Sub-option B: Formula** - computes the volume size from the machine's resources:

```
volume_size = volume_base + max(vCPUs / 2, memory_GiB / 8) * volume_factor
```

Where the formula values come from:

| Value | Source |
|-------|--------|
| `volume_base` | Configurable base ammount per landscape |
| `vCPUs` | Machine type metadata from the provider (e.g., 32 vCPUs for `m6i.8xlarge`) |
| `memory_GiB` | Machine type metadata from the provider (e.g., 128 GiB for `m6i.8xlarge`) |
| `volume_factor` | Normalized resource multiplier for total volume size set per landscape |

**Example producing 148 GiB** for a 32 vCPU / 128 GiB RAM machine (e.g., `m6i.8xlarge`) on a GardenLinux landscape (`volume_base=20`, `volume_factor=8`):

```
volume_size = 20 + max(32/2, 128/8) * 8
           = 20 + max(16, 16) * 8
           = 20 + 16 * 8
           = 20 + 128
           = 148 GiB
```

The computed volume size is shown alongside vCPU and memory in the machine type display name in the BTP cockpit, e.g. `m6i.8xlarge (32 vCPU, 128 GiB RAM, 148 GiB volume)`.

**BTP cockpit - main worker pool:**

![Approach 2 - dynamic volume size in BTP cockpit main worker pool](../../assets/adr-003-volume-size-approach2-dynamic-main-worker-pool.png)

**BTP cockpit - additional worker node pool:**

![Approach 2 - dynamic volume size in BTP cockpit additional worker node pool](../../assets/adr-003-volume-size-approach2-dynamic-additional-worker-pool.png)

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

### Billing

- The computed base volume size is included in the base machine price at no extra cost.
- Only the `additionalVolumeGb` amount is charged, per GB, per node, per month.
- The volume size is available as a status attribute on the Kubernetes node API, so the actual disk size per node can be detected at runtime for billing purposes.
- The price calculator must be updated to include a `volumeSizeGb` input field showing the additional cost when the value exceeds the default.

### Pros

- Large machines automatically get a larger disk without any user action.
- Users pay extra only for GB above the machine type default.

### Cons

- More complex to implement.
- Different formula parameters may be needed per OS image (GardenLinux vs. Ubuntu Pro).
- Users may not notice that the volume size differs per machine type.
- Every formula change requires KEB operators to be notified and ERS to be refreshed.
- There is a risk of temporarily inconsistent disk sizes displayed in the BTP cockpit if KEB already uses a different size for a given machine, since ERS refresh takes time.


## Decision

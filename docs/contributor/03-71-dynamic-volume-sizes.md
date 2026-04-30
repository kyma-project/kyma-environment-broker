<!--{"metadata":{"publish":true}}-->

# Dynamic Volume Sizes

## Overview

By default, the node volume size for every plan is a static value defined in the plan configuration. When the Dynamic Volume Sizes feature is enabled, Kyma Environment Broker (KEB) reads the volume size per machine type from the KCR (Kyma Consumption Reporter) ConfigMap instead, so that larger machines automatically receive appropriately sized disks.

The feature is controlled by two environment variables:

| Variable | Default | Description |
|---|---|---|
| `APP_BROKER_DYNAMIC_VOLUME_SIZE_ENABLED` | `false` | Enables dynamic volume size lookup. When `false`, the static plan default is used. |
| `APP_BROKER_KCR_CONFIG_MAP_NAME` | `consumption-reporter-config` | Name of the ConfigMap in the `kcp-system` namespace that provides the volume sizes. |

> ### Note:
> This feature is not applied to SAP Converged Cloud (OpenStack) plans, which do not configure node volumes.

## Behavior

### Provisioning

When a new runtime is provisioned, the volume size for the Kyma worker pool and all additional worker pools is read from the ConfigMap for each machine type.

### Update

When an existing runtime is updated:

- **Kyma worker pool**: if the machine type changes, the new volume size is read from the ConfigMap and applied. If the machine type is unchanged, the existing volume is preserved.
- **Additional worker pools**: the ConfigMap is consulted only for pools where the machine type is new or changed compared to the previous operation. Unchanged pools preserve their existing volume.

## Error Handling

KEB reads the ConfigMap on every provisioning and update operation — there is no caching, so configuration changes take effect without a restart.

| Condition | Result |
|---|---|
| Kubernetes API error reading the ConfigMap | Operation retried (temporary error) |
| Machine type not found in the ConfigMap | Operation failed (permanent error) |

## Startup Validation

When `APP_BROKER_DYNAMIC_VOLUME_SIZE_ENABLED` is `true`, KEB reads the ConfigMap at startup and verifies that every machine type in the providers configuration (AWS, Azure, GCP, Alicloud) has a valid entry. If any machine types are missing, KEB exits with a fatal error that lists all missing entries at once.

<!--{"metadata":{"requirement":"RECOMMENDED","type":"INTERNAL","category":"FEATURE"}}-->

# Updating Kyma Environment Broker: Dynamic Node Volume Sizes

> ### Note:
> No action is required. This notable change is informational only. The feature is disabled by default and will be enabled at a later stage.

## What's Changed

KEB now supports reading node volume sizes per machine type from the KCR (Kyma Consumption Reporter) ConfigMap instead of using static per-plan defaults. This ensures that larger machines automatically receive appropriately sized disks.

The feature is controlled by a new environment variable:

| Variable | Default | Description |
|---|---|---|
| `APP_BROKER_DYNAMIC_VOLUME_SIZE_ENABLED` | `false` | Enables dynamic volume size lookup. |

SAP Converged Cloud (OpenStack) plans are not affected — they do not configure node volumes.

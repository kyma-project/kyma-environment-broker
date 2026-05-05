<!--{"metadata":{"requirement":"RECOMMENDED","type":"INTERNAL","category":"FEATURE"}}-->

# Updating Kyma Environment Broker: Dynamic Node Volume Sizes

> ### Note:
> No action is required. This notable change is informational only. The feature is disabled by default and will be enabled at a later stage.

## What's Changed

KEB now supports reading node volume sizes per machine type from the Kyma Consumption Reporter (KCR) ConfigMap instead of using static per-plan defaults. This ensures that larger machines automatically receive appropriately sized disks.

The feature is controlled by a new environment variable:

| Variable | Default | Description |
|---|---|---|
| **APP_BROKER_DYNAMIC_VOLUME_SIZE_ENABLED** | `false` | Enables dynamic volume size lookup. |

By default, the dynamic node volume sizes feature is disabled. Once KCR and the cluster volume migration tooling are ready, another notable change will be published to inform you that you must enable the feature. The migration tool will be used to migrate existing clusters to the new machine-type-appropriate volume sizes.

Regardless of the `APP_BROKER_DYNAMIC_VOLUME_SIZE_ENABLED` flag, an explicit 80 GiB node volume is now introduced for new SAP Cloud Infrastructure (OpenStack) runtimes. Previously, SAP Cloud Infrastructure workers had no volume configured.

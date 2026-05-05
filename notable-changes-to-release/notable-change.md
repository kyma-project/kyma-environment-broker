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

The feature is disabled by default. Once KCR and the cluster volume migration tooling are ready, a separate mandatory notable change will be published to enable it. The migration tool will be used to migrate existing clusters to the new machine-type-appropriate volume sizes.

Additionally, new SAP Converged Cloud (OpenStack) runtimes now always receive an explicit 80 GiB node volume. Previously, SAP Converged Cloud workers had no volume configured. This change takes effect regardless of the `APP_BROKER_DYNAMIC_VOLUME_SIZE_ENABLED` flag. The default volume size is now also configured via the `volumeSizeGb` field in the SAP Converged Cloud plan configuration, consistent with other plans:

```yaml
sap-converged-cloud:
  volumeSizeGb: 80
```

<!--{"metadata":{"requirement":"RECOMMENDED","type":"INTERNAL","category":"FEATURE"}}-->

# Updating Kyma Environment Broker: Additional Node Volume Size Parameter

> ### Note:
> No action is required. This notable change is informational only. The feature is disabled by default and must be explicitly enabled per plan.

## What's Changed

KEB now supports an optional `additionalVolumeSizeGi` parameter that lets users request extra disk space on top of the default node volume size. The total volume allocated to a worker node equals the default size (from the KCR ConfigMap when dynamic volumes are enabled, or the plan static default) plus `additionalVolumeSizeGi`.

The parameter is available on both the main Kyma worker pool and on each entry in `additionalWorkerNodePools`. Its value persists across subsequent updates — a request that only changes the machine type will still include the previously set extra space.

The feature is controlled by a new environment variable:

| Variable | Default | Description |
|---|---|---|
| `APP_BROKER_ADDITIONAL_VOLUME_GI_PLANS` | _(empty)_ | Comma-separated list of plan names for which the `additionalVolumeSizeGi` parameter is exposed in the schema. Leave empty to disable the feature. Requires `APP_BROKER_DYNAMIC_VOLUME_SIZE_ENABLED=true`. |

When the list is empty (default), the `additionalVolumeSizeGi` field is not present in the plan schema and any attempt to pass it is rejected.

<!--{"metadata":{"requirement":"RECOMMENDED","type":"INTERNAL","category":"CONFIGURATION"}}-->

# KEB: Azure Availability Zone Discovery

> ### Note:
> No action is required to keep the existing behavior. Zone discovery for Microsoft Azure is opt-in and disabled by default. Enable it only if you want KEB to determine available zones dynamically from the Azure ResourceSKUs API instead of using static zone assignments.

> ### Caution:
> Enabling `zonesDiscovery` for Azure requires the Service Principal in the Gardener Secret to have the `Microsoft.Compute/skus/read` permission on the target subscription. The Contributor role alone may not be sufficient, depending on how it is configured. If the Azure zone cache fails to fill on startup, it returns errors such as `AuthorizationFailed` on `Microsoft.Compute/skus` in KEB logs, and KEB falls back to live API calls per request, which also fail with the same error.
> To recover, disable `zonesDiscovery` for Azure and investigate the subscription permissions.

## What's Changed

KEB now supports dynamic availability zone discovery for Microsoft Azure, consistent with the existing AWS behavior.

When enabled, KEB queries the Azure ResourceSKUs API at provisioning and update time to determine which availability zones are available for the requested machine type and subscription. Zones with Azure-level restrictions (`restrictions[type=Zone]`) are automatically excluded.

Key behaviors:

- The Kyma worker node pool requires at least 3 available zones. Provisioning is rejected synchronously if the machine type does not meet this requirement in the requested region.
- Additional worker node pools require 3 zones for high-availability (HA) pools, or at least 1 zone for non-HA pools.
- If the same machine type appears in multiple worker pools, the API is called only once per unique machine type.
- A global background cache is populated at KEB startup and refreshed every hour, eliminating per-request latency for HTTP validation.
- If both static zone configuration and `zonesDiscovery: true` are provided, a warning is logged and static zones are ignored.

## Procedure

To enable Azure zone discovery, set `zonesDiscovery: true` in the `providersConfig` ConfigMap under the `azure` section:

```yaml
azure:
  zonesDiscovery: true
```

Restart KEB after applying the configuration change.

## Post-Update Steps

Monitor KEB startup logs for cache fill confirmation:

```json
{"level":"INFO","msg":"Azure zone cache filled for region westeurope (24 machine types)"}
```

If credentials are unavailable at startup, KEB falls back to per-call mode and logs a warning. Zone discovery still works, but without the latency benefit of the global cache. Cache refreshes at one-hour intervals.

<!--{"metadata":{"requirement":"RECOMMENDED","type":"INTERNAL","category":"CONFIGURATION"}}-->

# KEB: Azure Cloud Environment Configuration for Zone Discovery

> ### Note:
> No action is required to keep the existing behavior. When `zonesDiscovery: true` is set for Azure and `clientConfiguration` isn't provided, KEB automatically detects the Azure cloud environment at startup. Setting `clientConfiguration` explicitly is recommended for production deployments.

## What's Changed

KEB now supports configuring the Azure cloud environment used for zone discovery. This change affects deployments where Azure zone discovery (`zonesDiscovery: true`) is enabled.

Previously, KEB always used Azure Public Cloud endpoints for zone discovery API calls, which caused failures in restricted markets such as Azure China (Mooncake), where different API endpoints and token issuers are required.

KEB now supports the following Azure cloud environments:

| Value | Environment |
|---|---|
| `public` | Azure Public Cloud |
| `china` | Azure China (Mooncake) |
| `usgov` | Azure US Government |

When **clientConfiguration** isn't set, KEB auto-discovers the correct environment at startup by probing each cloud in order (`public`, `china`, `usgov`) using a single OAuth token request per candidate. The result is determined before the HTTP server starts accepting requests. If all probes fail, KEB doesn't start.

## Procedure

To configure the Azure cloud environment explicitly, set **clientConfiguration** in the `providersConfig` ConfigMap under the `azure` section:

```yaml
azure:
  zonesDiscovery: true
  clientConfiguration: china
```

Restart KEB after applying the configuration change.

## Post-Update Steps

Verify the correct cloud environment is used by checking KEB startup logs.

When using explicit configuration, confirm the startup logs contain the following entries:

```json
{"level":"INFO","msg":"Azure cloud configured explicitly","cloud":"china"}
{"level":"INFO","msg":"Azure zone cache filled (12/12 regions)"}
```

When using auto-discovery, confirm the startup logs contain the following entries:

```json
{"level":"INFO","msg":"Azure cloud probe failed","cloud":"public"}
{"level":"INFO","msg":"Azure cloud auto-discovered","cloud":"china"}
{"level":"INFO","msg":"Azure zone cache filled (12/12 regions)"}
```

## Related Documentation

- [Zones Discovery](https://github.com/kyma-project/kyma-environment-broker/blob/main/docs/contributor/03-55-zones-discovery.md)

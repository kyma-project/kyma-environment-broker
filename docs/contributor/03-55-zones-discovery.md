<!--{"metadata":{"publish":true}}-->

# Zones Discovery

## Overview

The Zones Discovery feature extends Kyma Environment Broker (KEB) to dynamically determine availability zones for both the Kyma worker node pool and additional worker node pools during provisioning and updates.
Operators can configure worker node pools to use either static zone assignments (predefined in the configuration) or dynamic zone assignments (queried live from the hyperscaler).

This feature is supported on **AWS** and **Azure**.

Configuration:

```yaml
providersConfiguration:
  aws:
    zonesDiscovery: true
  azure:
    zonesDiscovery: true
```

If both a static configuration and **zonesDiscovery** are provided, a warning is logged on KEB's start to indicate that static zones are ignored.

Example log entries:

```json lines
{"level":"WARN", "msg":"Provider aws has zones discovery enabled, but region us-west-2 is configured with 4 static zone(s), which will be ignored."} 
{"level":"WARN", "msg":"Provider azure has zones discovery enabled, but region westeurope is configured with 3 static zone(s), which will be ignored."}
```

## Credentials

KEB resolves hyperscaler credentials from Gardener secrets referenced by a `CredentialsBinding`. The secret fields follow the Gardener convention for each provider type.

## Validation

During provisioning and updates, KEB validates the worker node pool configuration by retrieving a random hyperscaler subscription secret from Gardener and using it to query the available zones for the specified machine type.
- The Kyma worker node pool must support at least three zones.
- Additional worker node pools must support three zones if configured for high availability, or at least one zone otherwise.

To optimize performance, if the same machine type is used in multiple worker node pools, KEB queries the hyperscaler only once per unique machine type and reuses the result across all occurrences. This solution eliminates unnecessary duplicate calls.
The subscription secret is used only for validation. Its name is logged to support traceability in case of validation failures.

### Azure — zone restrictions

Azure `ResourceSKUs` API returns zone-level restrictions (`restrictions[type=Zone]`) which indicate that a given machine type is not available in a specific zone for the subscription. KEB automatically excludes restricted zones from the available zone list.

## Zones Discovery

If **zonesDiscovery** is enabled, KEB performs the `Discover_Available_Zones` step using hyperscaler credentials from the subscription secret resolved in the `Resolve_Subscription_Secret` step.
During provisioning, KEB queries all available zones for each unique machine type across both the Kyma worker node pool and additional worker node pools. During updates, it queries zones only for the additional worker node pools.
As in the validation process, the discovery mechanism guarantees that each unique machine type is queried only once, even if it is referenced by multiple worker node pools.
The results are stored in the operation under **operation.DiscoveredZones** as a mapping of machine types to zone lists. All discovered zones are logged.

## Discovered Zones Usage

When creating a Runtime resource, the Kyma worker node pool uses zones from the discovery results, limited to the required number.
During updates, existing worker node pools retain their current zones, while new worker pools use the discovered ones. This ensures consistent behavior and prevents re-randomization of zones for already provisioned pools.
Final assignments for both the Kyma worker node pool and additional worker node pools are logged.

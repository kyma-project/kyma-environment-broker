<!--{"metadata":{"requirement":"RECOMMENDED","type":"INTERNAL","category":"CONFIGURATION"}}-->

# KEB: Version-Agnostic Machine Type Support

> ### Note:
> This is a recommended configuration update. With version-agnostic machine types, your customers maintain stable configurations and benefit from automatic improvements to instance families without manual schema changes or service disruptions. To simplify generation upgrades in the future, update your Kyma Environment Broker (KEB) ConfigMap configuration.

## What's Changed

KEB now supports version-agnostic machine type names. Instead of configuring provider-specific instance names such as `m6i.large` (AWS) directly, you can use logical names such as `mi.large` that KEB resolves to the current optimized instance family at provisioning and update time. This change reduces coupling between machine names and hyperscaler generations. When a hyperscaler deprecates a machine generation or introduces a newer one, only the `machinesVersions` mapping in the KEB ConfigMap needs updating. The machine names exposed to users remain stable, while KEB applies the updated mapping to new worker pools automatically.

Versioned machine types such as `m6i.large` are still supported. However, we recommend marking them as deprecated in the schema and switching to the corresponding version-agnostic names.

## Procedure

1. Add the new version-agnostic machine names to the `machines` section.
2. For each provider, add a new **machinesVersions** field in `providersConfiguration`, and use it to map version-agnostic name patterns to concrete provider-specific instance name patterns applying `{placeholder}` syntax.

    > ### Note
    > To avoid recreating customers' Kubernetes nodes, map the version-agnostic machine types to the currently used versioned machine types. For example, `mi.large` to `m6i.large`.

1. Update the ERS registry to enable the version-agnostic machines for customers.
2. Deprecate the versioned names, but keep them in the schema until no instances reference them.

    > ### Note
    > Follow your usual deprecation procedure to ensure customers have enough time to apply the new configuration.

5. After all runtimes are updated by customers, remove deprecated entries from the schema.
6. Update the ERS registry to stop displaying the removed machine types.

    > ### Warning
    > If an update request includes a machine type that has already been removed from the schema, the BTP Cockpit form view shows the machine type field as empty. To submit the update, the user must switch to JSON view, then switch back to the form view. The `machineType` field is automatically reset to the first available value in the schema, and the update can be submitted.

### Example

See an example configuration for AWS:

```yaml
providersConfiguration:
  aws:
    machines:
      mi.large: mi.large (2vCPU, 8GB RAM)
      # Deprecated — kept for backward compatibility
      m6i.large: m6i.large (deprecated, use mi.large)
    machinesVersions:
      mi.{size}: m6i.{size}
```

With this configuration, KEB resolves `mi.large` to `m6i.large` before writing the Runtime CR. If the mapping is later updated to point to `m7i.{size}`, existing users continue to use `mi.large` without any change on their end.

For full provider-specific configuration examples and machine version resolution tables, see [Machines Versions](version-agnostic-machine-types.md).

## Post-Update Steps

KEB does not automatically reconcile existing worker pools when the **machinesVersions** configuration changes. When a machine generation update is required, work with SRE to update Runtime custom resources during a scheduled maintenance window using the Cluster Orchestrator.

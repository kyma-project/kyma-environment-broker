# Configure Machine Versions

This document describes how to introduce version-agnostic machine type names in Kyma Environment Broker (KEB) and migrate away from versioned names. For background on how machine type resolution works, see [Version-Agnostic Machine Types](version-agnostic-machine-types.md).

## Procedure

1. Add the new version-agnostic machine names to the `machines` section.
2. For each provider, add a new **machinesVersions** field in `providersConfiguration`, and use it to map version-agnostic name patterns to concrete provider-specific instance name patterns applying `{placeholder}` syntax.

    > ### Note
    > To avoid recreating customers' Kubernetes nodes, it's recommended to map the version-agnostic machine types to the currently used versioned machine types. For example, `mi.large` to `m6i.large`.

3. Update the ERS registry to enable the version-agnostic machines for customers.
4. Deprecate the versioned names, but keep them in the schema until no instances reference them. Follow your usual deprecation procedure to ensure customers have enough time to apply the new configuration.
5. After all runtimes are updated by customers, remove deprecated entries from the schema.
6. Update the ERS registry to stop displaying the removed machine types to customers.

    > ### Warning
    > If an update request includes a machine type that has already been removed from the schema, the BTP Cockpit form view shows the machine type field as empty. To submit the update, the user must switch to JSON view, then switch back to the form view. The `machineType` field is automatically reset to the first available value in the schema, and the update can be submitted.

## Post-Update Steps

KEB does not automatically reconcile existing worker pools when the **machinesVersions** configuration changes. When a machine generation update is required, work with SRE to update Runtime custom resources during a scheduled maintenance window using the Cluster Orchestrator.

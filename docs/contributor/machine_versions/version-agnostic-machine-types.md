# Version-Agnostic Machine Types

Using provider-specific versioned machine types, such as `m6i.large`, `Standard_D2s_v5`, or `n2-standard-2`, creates a tight coupling between a machine type and a specific generation. This causes the following issues:

- Adopting a newer machine generation requires schema changes.
- Older generations must be supported for backward compatibility.

To reduce this coupling, machine names are partially abstracted. Instead of requiring full provider-specific instance names, the configuration uses version-agnostic names such as `mi.large` (AWS) that Kyma Environment Broker (KEB) resolves to the current provider-specific instance at provisioning and update time.

When a hyperscaler deprecates a machine generation or introduces a newer one, only the `machinesVersions` mapping in the KEB ConfigMap needs updating. The machine names exposed to users remain stable, while KEB applies the updated mapping to new worker pools automatically.

> ### Tip:
> Versioned machine types such as `m6i.large` are still supported. If an input machine type does not match any configured mapping pattern, it is preserved as-is. However, we recommend marking versioned names as planned for deprecation in the schema and switching to the corresponding version-agnostic names.

> ### Note
> The concrete version a version-agnostic name resolves to may not always be the latest available generation. Operational constraints such as limited regional availability or storage class incompatibilities can require the mapping to target an older generation until those limitations are resolved. Additionally, some machine types may be introduced as specific concrete instances rather than version-agnostic types, depending on provider constraints or cost considerations. Before adopting version-agnostic machine types for a provider, consult the respective hyperscaler's documentation directly. When a mapping is updated, review the change for potential breaking implications, such as storage class or regional availability changes, before applying it.

## Machine Type Resolution

Machine type resolution is applied in the following cases:

- In the AWS client, when fetching availability zones.
- During provisioning of the Kyma worker node pool.
- During provisioning of additional worker node pools.
- During updates of the Kyma worker node pool, if the machine type is changed.
- During creation of new additional worker node pools as part of an update.
- During updates of existing additional worker node pools, if the machine type is changed.

> ### Note:
> KEB does not automatically reconcile or update existing worker node pools when the machines versions configuration changes.
> For example, if a user updates administrators after the machines versions configuration has changed, existing worker pools are not updated automatically and nodes are not restarted.
> This behavior is intentional, to avoid unnecessary disruption, especially during periods of peak load.

### How Resolution Works

Resolution is based on pattern matching:

1. The configured machine type is compared against the templates in **machinesVersions**.
2. If a template matches, its placeholders are substituted into the mapped output template.
3. If no template matches, the original value is returned unchanged.

This allows version-agnostic names such as `mi.large` to resolve to the current provider-specific instance names, while still accepting explicit values.

For steps on setting up and migrating to version-agnostic machine types, see [Configure Machine Versions](configure-machine-versions.md).

## Version-Agnostic Machine Types by Provider

Version-agnostic machine types are currently only available for Amazon Web Services.

> ### Note
> The names shown in this section are the recommended conventions used in the default KEB configuration. You can define your own version-agnostic names as long as they follow the `{placeholder}` pattern syntax and do not conflict with existing machine type names in the schema.

### Amazon Web Services

AWS version-agnostic types use short alphabetic family prefixes without an explicit generation number. The `{size}` placeholder matches any standard size suffix such as `large`, `xlarge`, or `16xlarge`.

| Version-Agnostic Prefix |      Purpose      | Resolves To |
|:-----------------------:|:-----------------:|:-----------:|
|          `mi`           |  General-purpose  |    `m6i`    |

Configuration:

```yaml
providersConfiguration:
  aws:
    machines:
      # Version-agnostic machines
      mi.large: mi.large (2vCPU, 8GB RAM)

      # Versioned machines planned for deprecation
      m5.large: m5.large (planned for deprecation, use mi.large)
      m6i.large: m6i.large (planned for deprecation, use mi.large)

    machinesVersions:
      mi.{size}: m6i.{size}
      m5.{size}: m6i.{size}
```

Machine version resolution:

|     Input     | Input Template | Output Template |    Output     |
|:-------------:|:--------------:|:---------------:|:-------------:|
|  `mi.large`   |  `mi.{size}`   |  `m6i.{size}`   |  `m6i.large`  |
|  `m5.large`   |  `m5.{size}`   |  `m6i.{size}`   |  `m6i.large`  |
|  `m6i.large`  |      `-`       |       `-`       |  `m6i.large`  |

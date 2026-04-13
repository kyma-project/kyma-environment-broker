# Machines Versions

Machine types can be specified either as concrete hyperscaler instance names, such as `m6i.large`, `Standard_D2s_v5`, or `n2-standard-2`, or as version-agnostic names.

Using provider-specific instance names directly creates a tight coupling between a machine type and a specific generation. This causes the following issues:

- Adopting a newer machine generation requires schema changes.
- Older values must still be supported for backward compatibility.

To reduce this coupling, machine names can be partially abstracted. Instead of requiring full provider-specific instance names, the configuration can use version-agnostic names. 
The concrete instance name is then resolved through a version mapping.

If an input machine type does not match any configured mapping pattern, it is preserved as-is.

## Machine Type Resolution

Machine type resolution is applied in the following cases:
- In the AWS client, when fetching availability zones.
- During provisioning of the Kyma worker node pool.
- During provisioning of any additional worker node pools.
- During updates of the Kyma worker node pool, if the machine type is changed.
- During creation of new additional worker node pools as part of an update.
- During updates of existing additional worker node pools, if the machine type is changed.

> ### Note:
> Kyma Environment Broker (KEB) does not automatically reconcile or update existing worker pools when the `machinesVersions` configuration changes.
> For example, if a user updates administrators after the `machinesVersions` configuration has changed, existing worker pools are not updated automatically and nodes are not restarted.
> This behavior is intentional, to avoid unnecessary disruption, especially during periods of peak load.

## Example Configuration

```yaml
providersConfiguration:
  aws:
    machines:
      # Version Agnostic machines
      mi.large: mi.large (2vCPU, 8GB RAM)
      ci.large: ci.large (2vCPU, 4GB RAM)
      ri.large: ri.large (2vCPU, 16GB RAM)
      ii.large: ii.large (2vCPU, 16GB RAM)
      g.xlarge: g.xlarge (1GPU, 4vCPU, 16GB RAM)*
      gdn.xlarge: gdn.xlarge (1GPU, 4vCPU, 16GB RAM)*
      
      # Machines with explicit version
      m5.large: m5.large (deprecated, use mi.large)
      m6i.large: m6i.large (deprecated, use mi.large)
      c7i.large: c7i.large (deprecated, use ci.large)
      g6.xlarge: g6.xlarge (deprecated, use g.xlarge)*
      g4dn.xlarge: g4dn.xlarge (deprecated, use gdn.xlarge)*

    machinesVersions:
      mi.{size}: m6i.{size}
      ci.{size}: c7i.{size}
      g.{size}: g6.{size}
      gdn.{size}: g4dn.{size}
      ri.{size}: r8i.{size}
      ii.{size}: i7i.{size}
      m5.{size}: m6i.{size}
```

## How Resolution Works

Resolution is based on pattern matching:

1. The configured machine type is compared against the templates in `machinesVersions`.
2. If a template matches, its placeholders are substituted into the mapped output template.
3. If no template matches, the original value is returned unchanged.

This allows version-agnostic names such as `mi.large` to resolve to the current provider-specific instance names, while still accepting explicit values.

### Resolution Examples

|     Input     | Input Template | Output Template |    Output     |
|:-------------:|:--------------:|:---------------:|:-------------:|
|  `mi.large`   |  `mi.{size}`   |  `m6i.{size}`   |  `m6i.large`  |
|  `ci.large`   |  `ci.{size}`   |  `c7i.{size}`   |  `c7i.large`  |
|  `ri.large`   |  `ri.{size}`   |  `r8i.{size}`   |  `r8i.large`  |
|  `ii.large`   |  `ii.{size}`   |  `i7i.{size}`   |  `i7i.large`  |
|  `g.xlarge`   |   `g.{size}`   |   `g6.{size}`   |  `g6.xlarge`  |
| `gdn.xlarge`  |  `gdn.{size}`  |  `g4dn.{size}`  | `g4dn.xlarge` |
|  `m5.large`   |  `m5.{size}`   |  `m6i.{size}`   |  `m6i.large`  |
|  `m6i.large`  |      `-`       |       `-`       |  `m6i.large`  |
|  `c7i.large`  |      `-`       |       `-`       |  `c7i.large`  |
|  `g6.xlarge`  |      `-`       |       `-`       |  `g6.xlarge`  |
| `g4dn.xlarge` |      `-`       |       `-`       | `g4dn.xlarge` |

# Version-Agnostic Machine Types

Using provider-specific versioned machine types, such as `m6i.large`, `Standard_D2s_v5`, or `n2-standard-2`, creates a tight coupling between a machine type and a specific generation. This causes the following issues:

- Adopting a newer machine generation requires schema changes.
- Older generations must be supported for backward compatibility.

To reduce this coupling, machine names are partially abstracted. Instead of requiring full provider-specific instance names, the configuration uses version-agnostic names such as `mi.large` (AWS) that Kyma Environment Broker (KEB) resolves to the current provider-specific instance at provisioning and update time.

When a hyperscaler deprecates a machine generation or introduces a newer one, only the `machinesVersions` mapping in the KEB ConfigMap needs updating. The machine names exposed to users remain stable, while KEB applies the updated mapping to new worker pools automatically.

> ### Tip:
> Versioned machine types such as `m6i.large` are still supported. If an input machine type does not match any configured mapping pattern, it is preserved as-is. However, we recommend marking versioned names as deprecated in the schema and switching to the corresponding version-agnostic names.

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

Resolution is based on the following pattern matching:

1. The configured machine type is compared against the templates in **machinesVersions**.
2. If a template matches, its placeholders are substituted into the mapped output template.
3. If no template matches, the original value is returned unchanged.

This allows version-agnostic names such as `mi.large` to resolve to the current provider-specific instance names, while still accepting explicit values.

For steps on setting up and migrating to version-agnostic machine types, see [Configure Machine Versions](configure-machine-versions.md).

## Version-Agnostic Machine Types by Provider

Each supported provider uses a pattern-based naming scheme. The **machinesVersions** field maps each version-agnostic name pattern to the concrete instance type currently in use. Deprecated explicit-version types remain in the schema for backward compatibility.

> ### Note
> The names shown in this section are the recommended conventions used in the default KEB configuration. You can define your own version-agnostic names as long as they follow the `{placeholder}` pattern syntax and do not conflict with existing machine type names in the schema.

### Amazon Web Services

AWS version-agnostic types use short alphabetic family prefixes without an explicit generation number. The `{size}` placeholder matches any standard size suffix such as `large`, `xlarge`, or `16xlarge`.

| Version-Agnostic Prefix | Purpose           | Resolves To |
|:-----------------------:|:-----------------:|:-----------:|
| `mi`                    | General-purpose   | `m6i`       |
| `ci`                    | Compute-optimized | `c7i`       |
| `ri`                    | Memory-optimized  | `r8i`       |
| `ii`                    | Storage-optimized | `i7i`       |
| `g`                     | GPU (G6 family)   | `g6`        |
| `gdn`                   | GPU (G4dn family) | `g4dn`      |

Configuration:

```yaml
providersConfiguration:
  aws:
    machines:
      # Version-agnostic machines
      mi.large: mi.large (2vCPU, 8GB RAM)
      ci.large: ci.large (2vCPU, 4GB RAM)
      ri.large: ri.large (2vCPU, 16GB RAM)
      ii.large: ii.large (2vCPU, 16GB RAM)
      g.xlarge: g.xlarge (1GPU, 4vCPU, 16GB RAM)*
      gdn.xlarge: gdn.xlarge (1GPU, 4vCPU, 16GB RAM)*

      # Deprecated machines with explicit version
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


Machine version resolution:

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

### Microsoft Azure

Azure version-agnostic types omit the `_v{N}` generation suffix. The `{size}` placeholder matches the numeric size component of the instance name.

| Version-Agnostic Pattern  | Purpose                          | Resolves To                  |
|:-------------------------:|:--------------------------------:|:----------------------------:|
| `Standard_D{size}s`       | General-purpose (premium storage)| `Standard_D{size}s_v5`       |
| `Standard_D{size}`        | General-purpose                  | `Standard_D{size}_v3`        |
| `Standard_F{size}s`       | Compute-optimized                | `Standard_F{size}s_v2`       |
| `Standard_NC{size}as_T4`  | GPU                              | `Standard_NC{size}as_T4_v3`  |
| `Standard_E{size}s`       | Memory-optimized                 | `Standard_E{size}s_v6`       |
| `Standard_L{size}s`       | Storage-optimized                | `Standard_L{size}s_v3`       |

Configuration:

```yaml
providersConfiguration:
  azure:
    machines:
      # Version-agnostic machines
      Standard_D2s: Standard_D2s (2vCPU, 8GB RAM)
      Standard_D4: Standard_D4 (4vCPU, 16GB RAM)
      Standard_F2s: Standard_F2s (2vCPU, 4GB RAM)
      Standard_NC4as_T4: Standard_NC4as_T4 (1GPU, 4vCPU, 28GB RAM)*
      Standard_E2s: Standard_E2s (2vCPU, 16GB RAM)
      Standard_L8s: Standard_L8s (8vCPU, 64GB RAM)

      # Deprecated machines with explicit version
      Standard_D2s_v5: Standard_D2s_v5 (deprecated, use Standard_D2s)
      Standard_D4_v3: Standard_D4_v3 (deprecated, use Standard_D4)
      Standard_F2s_v2: Standard_F2s_v2 (deprecated, use Standard_F2s)
      Standard_NC4as_T4_v3: Standard_NC4as_T4_v3 (deprecated, use Standard_NC4as_T4)*

    machinesVersions:
      Standard_D{size}s: Standard_D{size}s_v5
      Standard_D{size}: Standard_D{size}_v3
      Standard_F{size}s: Standard_F{size}s_v2
      Standard_NC{size}as_T4: Standard_NC{size}as_T4_v3
      Standard_E{size}s: Standard_E{size}s_v6
      Standard_L{size}s: Standard_L{size}s_v3
```

Machine version resolution:

|         Input          |      Input Template      |       Output Template       |         Output         |
|:----------------------:|:------------------------:|:---------------------------:|:----------------------:|
|     `Standard_D2s`     |   `Standard_D{size}s`    |   `Standard_D{size}s_v5`    |   `Standard_D2s_v5`    |
|     `Standard_D4`      |    `Standard_D{size}`    |    `Standard_D{size}_v3`    |    `Standard_D4_v3`    |
|     `Standard_F2s`     |   `Standard_F{size}s`    |   `Standard_F{size}s_v2`    |   `Standard_F2s_v2`    |
|  `Standard_NC4as_T4`   | `Standard_NC{size}as_T4` | `Standard_NC{size}as_T4_v3` | `Standard_NC4as_T4_v3` |
|     `Standard_E2s`     |   `Standard_E{size}s`    |   `Standard_E{size}s_v6`    |   `Standard_E2s_v6`    |
|     `Standard_L8s`     |   `Standard_L{size}s`    |   `Standard_L{size}s_v3`    |   `Standard_L8s_v3`    |
|   `Standard_D2s_v5`    |           `-`            |             `-`             |   `Standard_D2s_v5`    |
|    `Standard_D4_v3`    |           `-`            |             `-`             |    `Standard_D4_v3`    |
|   `Standard_F2s_v2`    |           `-`            |             `-`             |   `Standard_F2s_v2`    |
| `Standard_NC4as_T4_v3` |           `-`            |             `-`             | `Standard_NC4as_T4_v3` |

### Google Cloud

GCP version-agnostic types omit the generation number from the family prefix. The `{size}` placeholder matches the vCPU count in the instance name:

| Version-Agnostic Pattern | Purpose                       | Resolves To                        |
|:------------------------:|:-----------------------------:|:----------------------------------:|
| `n-standard-{size}`      | General-purpose               | `n2-standard-{size}`               |
| `cd-highcpu-{size}`      | Compute-optimized             | `c2d-highcpu-{size}`               |
| `g-standard-{size}`      | GPU                           | `g2-standard-{size}`               |
| `m-ultramem-{size}`      | Memory-optimized              | `m3-ultramem-{size}`               |
| `z-highmem-{size}-standardlssd`       | Storage-optimized (local SSD) | `z3-highmem-{size}-standardlssd`   |

Configuration:

```yaml
providersConfiguration:
  gcp:
    machines:
      # Version-agnostic machines
      n-standard-2: n-standard-2 (2vCPU, 8GB RAM)
      cd-highcpu-2: cd-highcpu-2 (2vCPU, 4GB RAM)
      g-standard-4: g-standard-4 (1GPU, 4vCPU, 16GB RAM)*
      m-ultramem-32: m-ultramem-32 (32vCPU, 976GB RAM)
      z-highmem-14-standardlssd: z-highmem-14-standardlssd (14vCPU, 112GB RAM)

      # Deprecated machines with explicit version
      n2-standard-2: n2-standard-2 (deprecated, use n-standard-2)
      c2d-highcpu-2: c2d-highcpu-2 (deprecated, use cd-highcpu-2)
      g2-standard-4: g2-standard-4 (deprecated, use g-standard-4)*

    machinesVersions:
      n-standard-{size}: n2-standard-{size}
      cd-highcpu-{size}: c2d-highcpu-{size}
      g-standard-{size}: g2-standard-{size}
      m-ultramem-{size}: m3-ultramem-{size}
      z-highmem-{size}-standardlssd: z3-highmem-{size}-standardlssd
```

Machine version resolution:

|      Input      |   Input Template    |         Output Template          |            Output            |
|:---------------:|:-------------------:|:--------------------------------:|:----------------------------:|
| `n-standard-2`  | `n-standard-{size}` |       `n2-standard-{size}`       |       `n2-standard-2`        |
| `cd-highcpu-2`  | `cd-highcpu-{size}` |       `c2d-highcpu-{size}`       |       `c2d-highcpu-2`        |
| `g-standard-4`  | `g-standard-{size}` |       `g2-standard-{size}`       |       `g2-standard-4`        |
| `m-ultramem-32` | `m-ultramem-{size}` |       `m3-ultramem-{size}`       |       `m3-ultramem-32`       |
| `z-highmem-14-standardlssd`  | `z-highmem-{size}-standardlssd`  | `z3-highmem-{size}-standardlssd` | `z3-highmem-14-standardlssd` |
| `n2-standard-2` |         `-`         |               `-`                |       `n2-standard-2`        |
| `c2d-highcpu-2` |         `-`         |               `-`                |       `c2d-highcpu-2`        |
| `g2-standard-4` |         `-`         |               `-`                |       `g2-standard-4`        |

### SAP Cloud Infrastructure

SAP Cloud Infrastructure machine types follow the `g_c{c_size}_m{m_size}` scheme, where `c_size` is the vCPU count and `m_size` is the memory in GB. These first-generation types do not carry an explicit version identifier. When **machinesVersions** is configured, KEB resolves them to the corresponding second-generation `_v2` variant.

| Version-Agnostic Pattern | Purpose         | Resolves To                |
|:------------------------:|:---------------:|:--------------------------:|
| `g_c{c_size}_m{m_size}`  | General-purpose | `g_c{c_size}_m{m_size}_v2` |

Configuration:

```yaml
providersConfiguration:
  sap-converged-cloud:
    machines:
      g_c2_m8: g_c2_m8 (2vCPU, 8GB RAM)
      g_c4_m16: g_c4_m16 (4vCPU, 16GB RAM)
      g_c8_m32: g_c8_m32 (8vCPU, 32GB RAM)
      g_c16_m64: g_c16_m64 (16vCPU, 64GB RAM)
      g_c32_m128: g_c32_m128 (32vCPU, 128GB RAM)
      g_c64_m256: g_c64_m256 (64vCPU, 256GB RAM)
    machinesVersions:
      g_c{c_size}_m{m_size}: g_c{c_size}_m{m_size}_v2
```

Machine version resolution:

|   Input   |     Input Template      |      Output Template       |    Output    |
|:---------:|:-----------------------:|:--------------------------:|:------------:|
| `g_c2_m8` | `g_c{c_size}_m{m_size}` | `g_c{c_size}_m{m_size}_v2` | `g_c2_m8_v2` |

### Alibaba Cloud

Alibaba Cloud version-agnostic types use the `ecs.gi.{size}` scheme, omitting the generation number from the family prefix.

| Version-Agnostic Pattern | Purpose         | Resolves To      |
|:------------------------:|:---------------:|:----------------:|
| `ecs.gi.{size}`          | General-purpose | `ecs.g9i.{size}` |

Configuration:

```yaml
providersConfiguration:
  alicloud:
    machines:
      # Version-agnostic machines
      ecs.gi.large: ecs.gi.large (2vCPU, 8GB RAM)
      ecs.gi.xlarge: ecs.gi.xlarge (4vCPU, 16GB RAM)
      ecs.gi.2xlarge: ecs.gi.2xlarge (8vCPU, 32GB RAM)

      # Deprecated machines with explicit version
      ecs.g9i.large: ecs.g9i.large (deprecated, use ecs.gi.large)
      ecs.g9i.xlarge: ecs.g9i.xlarge (deprecated, use ecs.gi.xlarge)

    machinesVersions:
      ecs.gi.{size}: ecs.g9i.{size}
```

Machine version resolution:

|      Input      | Input Template  | Output Template  |     Output      |
|:---------------:|:---------------:|:----------------:|:---------------:|
| `ecs.gi.large`  | `ecs.gi.{size}` | `ecs.g9i.{size}` | `ecs.g9i.large` |
| `ecs.g9i.large` |       `-`       |       `-`        | `ecs.g9i.large` |

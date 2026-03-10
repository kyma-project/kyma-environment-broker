# Current configuration

## AWS

```yaml
providersConfiguration:
  aws:
    machines:
      m6i.large: m6i.large (2vCPU, 8GB RAM)
      m6i.16xlarge: m6i.16xlarge (64vCPU, 256GB RAM)
      m5.large: m5.large (2vCPU, 8GB RAM)
      m5.16xlarge: m5.16xlarge (64vCPU, 256GB RAM)
      c7i.large: c7i.large (2vCPU, 4GB RAM)
      c7i.16xlarge: c7i.16xlarge (64vCPU, 128GB RAM)
      g6.xlarge: g6.xlarge (1GPU, 4vCPU, 16GB RAM)*
      g6.16xlarge: g6.16xlarge (1GPU, 64vCPU, 256GB RAM)*
      g4dn.xlarge: g4dn.xlarge (1GPU, 4vCPU, 16GB RAM)*
      g4dn.16xlarge: g4dn.16xlarge (1GPU, 64vCPU, 256GB RAM)*

      # New memory-intensive machine types
      r7i.large: r7i.large (2vCPU, 16GB RAM)
      r7i.16xlarge: r7i.16xlarge (64vCPU, 512GB RAM)
        
      # New storage-intensive machine types
      i7i.large: i7i.large (2vCPU, 16GB RAM)
      i7i.16xlarge: i7i.16xlarge (64vCPU, 512GB RAM)
```

## Azure

```yaml
providersConfiguration:
  azure:
    machines:
      Standard_D2s_v5: Standard_D2s_v5 (2vCPU, 8GB RAM)
      Standard_D64s_v5: Standard_D64s_v5 (64vCPU, 256GB RAM)
      Standard_D4_v3: Standard_D4_v3 (4vCPU, 16GB RAM)
      Standard_D64_v3: Standard_D64_v3 (64vCPU, 256GB RAM)
      Standard_F2s_v2: Standard_F2s_v2 (2vCPU, 4GB RAM)
      Standard_F64s_v2: Standard_F64s_v2 (64vCPU, 128GB RAM)
      Standard_NC4as_T4_v3: Standard_NC4as_T4_v3 (1GPU, 4vCPU, 28GB RAM)*
      Standard_NC64as_T4_v3: Standard_NC64as_T4_v3 (4GPU, 64vCPU, 440GB RAM)*

      # New memory-intensive machine types
      Standard_E2s_v6: Standard_E2s_v6 (2vCPU, 16GB RAM)
      Standard_E64s_v6: Standard_E64s_v6 (64vCPU, 512GB RAM)

      # New storage-intensive machine types
      Standard_L8s_v3: Standard_L8s_v3 (8vCPU, 64GB RAM)
      Standard_L64s_v3: Standard_L64s_v3 (64vCPU, 512GB RAM)
```

## GCP

```yaml
providersConfiguration:
  gcp:
    machines:
      n2-standard-2: n2-standard-2 (2vCPU, 8GB RAM)
      n2-standard-64: n2-standard-64 (64vCPU, 256GB RAM)
      c2d-highcpu-2: c2d-highcpu-2 (2vCPU, 4GB RAM)
      c2d-highcpu-56: c2d-highcpu-56 (56vCPU, 112GB RAM)
      g2-standard-4: g2-standard-4 (1GPU, 4vCPU, 16GB RAM)*
      g2-standard-48: g2-standard-48 (4GPU, 48vCPU, 192GB RAM)*

      # New memory-intensive machine types
      m3-ultramem-32: m3-ultramem-32 (32vCPU, 976GB RAM)
      m3-ultramem-64: m3-ultramem-64 (64vCPU, 1,952GB RAM)

      # New storage-intensive machine types
      z3-highmem-14-standardlssd: z3-highmem-14-standardlssd (14vCPU, 112GB RAM)
      z3-highmem-44-standardlssd: z3-highmem-44-standardlssd (44vCPU, 352GB RAM)
```

## SAP Cloud Infrastructure

```yaml
providersConfiguration:
  sap-converged-cloud:
    machines:
      g_c2_m8: g_c2_m8 (2vCPU, 8GB RAM)
      g_c64_m256: g_c64_m256 (64vCPU, 256GB RAM)
```

## Alibaba Cloud

```yaml
providersConfiguration:
    alicloud:
      machines:
        "ecs.g9i.large": "ecs.g9i.large (2vCPU, 8GB RAM)"
        "ecs.g9i.16xlarge": "ecs.g9i.16xlarge (64vCPU, 256GB RAM)"
```


# Semi-Abstract Configuration

To simplify upgrades between instance generations, the configuration can be partially abstracted.
Instead of referencing full instance family names, a logical machine type is used. The actual family is then resolved through a mapping.

## AWS

```yaml
providersConfiguration:
  aws:
    machines:
      mi.large: mi.large (2vCPU, 8GB RAM)
      mi.16xlarge: mi.16xlarge (64vCPU, 256GB RAM)
      m.large: m.large (2vCPU, 8GB RAM)
      m.16xlarge: m.16xlarge (64vCPU, 256GB RAM)
      ci.large: ci.large (2vCPU, 4GB RAM)
      ci.16xlarge: ci.16xlarge (64vCPU, 128GB RAM)
      g.xlarge: g.xlarge (1GPU, 4vCPU, 16GB RAM)*
      g.16xlarge: g.16xlarge (1GPU, 64vCPU, 256GB RAM)*
      gdn.xlarge: gdn.xlarge (1GPU, 4vCPU, 16GB RAM)*
      gdn.16xlarge: gdn.16xlarge (1GPU, 64vCPU, 256GB RAM)*

      # New memory-intensive machine types
      ri.large: ri.large (2vCPU, 16GB RAM)
      ri.16xlarge: ri.16xlarge (64vCPU, 512GB RAM)
        
      # New storage-intensive machine types
      ii.large: ii.large (2vCPU, 16GB RAM)
      ii.16xlarge: ii.16xlarge (64vCPU, 512GB RAM)

    machinesVersions:
      mi: m6i
      m: m5
      ci: c7i
      g: g6
      gdn: g4dn
      ri: r7i
      ii: i7i
```

### Resolution Logic

AWS instance types follow the format:

```
<family>.<size>
```

When using the semi-abstract configuration, the resolution process is:
1. Split the configured machine type by the dot (.) separator.
2. Extract the logical machine family (the part before the dot).
3. Look up the corresponding AWS instance family in `machinesVersions`.
4. Replace the logical family with the mapped AWS family.
5. Reconstruct the final AWS instance type.

### Example

```
Input:  mi.large

Step 1: mi | large
Step 2: lookup mi → m6i
Result: m6i.large
```

## Azure

```yaml
providersConfiguration:
  azure:
    machines:
      Standard_D2s: Standard_D2s (2vCPU, 8GB RAM)
      Standard_D64s: Standard_D64s (64vCPU, 256GB RAM)
      Standard_D4: Standard_D4 (4vCPU, 16GB RAM)
      Standard_D64: Standard_D64 (64vCPU, 256GB RAM)
      Standard_F2s: Standard_F2s (2vCPU, 4GB RAM)
      Standard_F64s: Standard_F64s (64vCPU, 128GB RAM)
      Standard_NC4as_T4: Standard_NC4as_T4 (1GPU, 4vCPU, 28GB RAM)*
      Standard_NC64as_T4: Standard_NC64as_T4 (4GPU, 64vCPU, 440GB RAM)*

      # New memory-intensive machine types
      Standard_E2s: Standard_E2s (2vCPU, 16GB RAM)
      Standard_E64s: Standard_E64s (64vCPU, 512GB RAM)

      # New storage-intensive machine types
      Standard_L8s: Standard_L8s (8vCPU, 64GB RAM)
      Standard_L64s: Standard_L64s (64vCPU, 512GB RAM)

    machinesVersions:
      Ds: v5
      D: v3
      Fs: v2
      NCas: v3
      Es: v6
      Ls: v3
```

### Resolution Logic

Azure instance types follow the format:

```
<prefix>_<machine family with size>_<optional additional info>_<version>
```

When using the semi-abstract configuration, the resolution process is:
1. Split the configured machine type by the underscore _.
2. Identify the logical machine family (second element) and remove numeric characters to isolate the family prefix.
3. Lookup the corresponding family version in `machinesVersions`.
4. Append the version to the input machine type.
5. Reconstruct the final Azure instance type.

### Example

```
Input: Standard_NC4as_T4

Step 1: Split → [Standard, NC4as, T4]
Step 2: Remove numbers from family → NCas
Step 3: Lookup version → NCas → v3
Step 4: Reconstruct → Standard_NC4as_T4_v3
```

## GCP

```yaml
providersConfiguration:
  gcp:
    machines:
      n-standard-2: n-standard-2 (2vCPU, 8GB RAM)
      n-standard-64: n-standard-64 (64vCPU, 256GB RAM)
      cd-highcpu-2: cd-highcpu-2 (2vCPU, 4GB RAM)
      cd-highcpu-56: cd-highcpu-56 (56vCPU, 112GB RAM)
      g-standard-4: g-standard-4 (1GPU, 4vCPU, 16GB RAM)*
      g-standard-48: g-standard-48 (4GPU, 48vCPU, 192GB RAM)*

      # New memory-intensive machine types
      m-ultramem-32: m-ultramem-32 (32vCPU, 976GB RAM)
      m-ultramem-64: m-ultramem-64 (64vCPU, 1,952GB RAM)

      # New storage-intensive machine types
      z-highmem-14-standardlssd: z-highmem-14-standardlssd (14vCPU, 112GB RAM)
      z-highmem-44-standardlssd: z-highmem-44-standardlssd (44vCPU, 352GB RAM)

    machinesVersions:
      n: n2
      cd: c2d
      g: g2
      m: m3
      z: z3
```

### Resolution Logic

AWS instance types follow the format:

```
<family with version>-<type>-<size>-<optional info>
```

When using the semi-abstract configuration, the resolution steps are:
1. Split the configured machine type by the - separator.
2. Identify the logical machine family (the first segment).
3. Map it to the corresponding GCP instance family using `machinesVersions`.
4. Replace the logical family with the mapped GCP family.
5. Reconstruct the final instance type string.

### Example

```
Input:  z-highmem-14-standardlssd

Step 1: Split → z | highmem | 14 | standardlssd
Step 2: Map z → z3
Step 3: Reconstruct → z3-highmem-14-standardlssd
```

## SAP Cloud Infrastructure

```yaml
  providersConfiguration:
    sap-converged-cloud:
      machines:
        g_c2_m8: g_c2_m8 (2vCPU, 8GB RAM)
        g_c64_m256: g_c64_m256 (64vCPU, 256GB RAM)
```

No machine types currently include version numbers in their names.

## Alibaba Cloud

```yaml
providersConfiguration:
  alicloud:
    machines:
      "ecs.gi.large": "ecs.gi.large (2vCPU, 8GB RAM)"
      "ecs.gi.16xlarge": "ecs.gi.16xlarge (64vCPU, 256GB RAM)"
    
    machinesVersions:
      gi: g9i
```

### Resolution Logic

Alibaba Cloud instance types follow the format:

```
<prefix>.<family with version>.<size>
```

When using the semi-abstract configuration, the resolution process is:
1. Split the configured machine type using the dot (.) separator.
2. Extract the logical machine family (the second segment).
3. Look up the corresponding Alibaba Cloud family version in `machinesVersions`.
4. Replace the logical family with the mapped versioned family.
5. Reconstruct the final instance type.

### Example

```
Input:  ecs.gi.large

Step 1: ecs | gi | large
Step 2: lookup gi → g9i
Result: ecs.g9i.large
```

# Abstract configuration

The abstract configuration fully separates logical machine types from actual instance types.
Instead of referencing instance families directly, machines are defined using logical categories such as `general`, `compute`, `memory`, `storage`, or `gpu`.

The actual instance types are defined in `machinesMapping`, which maps each logical machine to a concrete instance.
This allows instance generations or families to be changed by updating only the mapping, without modifying the main configuration.

## AWS

```yaml
providersConfiguration:
  aws:
    machines:
      general-2: general-2 (2vCPU, 8GB RAM)
      general-64: general-64 (64vCPU, 256GB RAM)
      general-prev-2: general-prev-2 (2vCPU, 8GB RAM)
      general-prev-64: general-prev-64 (64vCPU, 256GB RAM)
      compute-2: compute-2 (2vCPU, 4GB RAM)
      compute-64: compute-64 (64vCPU, 128GB RAM)
      gpu-4: gpu-4 (1GPU, 4vCPU, 16GB RAM)*
      gpu-64: gpu-64 (1GPU, 64vCPU, 256GB RAM)*
      gpu-legacy-4: gpu-legacy-4 (1GPU, 4vCPU, 16GB RAM)*
      gpu-legacy-64: gpu-legacy-64 (1GPU, 64vCPU, 256GB RAM)*

      # New memory-intensive machine types
      memory-2: memory-2 (2vCPU, 16GB RAM)
      memory-64: memory-64 (64vCPU, 512GB RAM)

      # New storage-intensive machine types
      storage-2: storage-2 (2vCPU, 16GB RAM)
      storage-64: storage-64 (64vCPU, 512GB RAM)

    machinesMapping:
      general-2: m6i.large
      general-64: m6i.16xlarge
      general-prev-2: m5.large
      general-prev-64: m5.16xlarge
      compute-2: c7i.large
      compute-64: c7i.16xlarge
      gpu-4: g6.xlarge
      gpu-64: g6.16xlarge
      gpu-legacy-4: g4dn.xlarge
      gpu-legacy-64: g4dn.16xlarge

      # New memory-intensive machine types
      memory-2: r7i.large
      memory-64: r7i.16xlarge

      # New storage-intensive machine types
      storage-2: i7i.large
      storage-64: i7i.16xlarge
```

## Azure

```yaml
providersConfiguration:
  azure:
    machines:
      general-2: general-2 (2vCPU, 8GB RAM)
      general-64: general-64 (64vCPU, 256GB RAM)
      general-prev-4: general-prev-4 (4vCPU, 16GB RAM)
      general-prev-64: general-prev-64 (64vCPU, 256GB RAM)
      compute-2: compute-2 (2vCPU, 4GB RAM)
      compute-64: compute-64 (64vCPU, 128GB RAM)
      gpu-4: gpu-4 (1GPU, 4vCPU, 28GB RAM)*
      gpu-64: gpu-64 (4GPU, 64vCPU, 440GB RAM)*

      # New memory-intensive machine types
      memory-2: memory-2 (2vCPU, 16GB RAM)
      memory-64: memory-64 (64vCPU, 512GB RAM)

      # New storage-intensive machine types
      storage-8: storage-8 (8vCPU, 64GB RAM)
      storage-64: storage-64 (64vCPU, 512GB RAM)

    machinesMapping:
      general-2: Standard_D2s_v5
      general-64: Standard_D64s_v5
      general-prev-4: Standard_D4_v3
      general-prev-64: Standard_D64_v3
      compute-2: Standard_F2s_v2
      compute-64: Standard_F64s_v2
      gpu-4: Standard_NC4as_T4_v3
      gpu-64: Standard_NC64as_T4_v3

      # New memory-intensive machine types
      memory-2: Standard_E2s_v6
      memory-64: Standard_E64s_v6

      # New storage-intensive machine types
      storage-8: Standard_L8s_v3
      storage-64: Standard_L64s_v3
```

## GCP

```yaml
providersConfiguration:
  gcp:
    machines:
      general-2: general-2 (2vCPU, 8GB RAM)
      general-64: general-64 (64vCPU, 256GB RAM)
      compute-2: compute-2 (2vCPU, 4GB RAM)
      compute-56: compute-56 (56vCPU, 112GB RAM)
      gpu-4: gpu-4 (1GPU, 4vCPU, 16GB RAM)*
      gpu-48: gpu-48 (4GPU, 48vCPU, 192GB RAM)*

      # New memory-intensive machine types
      memory-32: memory-32 (32vCPU, 976GB RAM)
      memory-64: memory-64 (64vCPU, 1,952GB RAM)

      # New storage-intensive machine types
      storage-14: storage-14 (14vCPU, 112GB RAM)
      storage-44: storage-44 (44vCPU, 352GB RAM)
        
    machinesMapping:
      general-2: n2-standard-2
      general-64: n2-standard-64
      compute-2: c2d-highcpu-2
      compute-56: c2d-highcpu-56
      gpu-4: g2-standard-4
      gpu-48: g2-standard-48

      # New memory-intensive machine types
      memory-32: m3-ultramem-32
      memory-64: m3-ultramem-64

      # New storage-intensive machine types
      storage-14: z3-highmem-14-standardlssd
      storage-44: z3-highmem-44-standardlssd
```

## SAP Cloud Infrastructure

```yaml
providersConfiguration:
  sap-converged-cloud:
    machines:
      general-2: general-2 (2vCPU, 8GB RAM)
      general-64: general-64 (64vCPU, 256GB RAM)

    machinesMapping:
      general-2: g_c2_m8
      general-64: g_c64_m256
```

## Alibaba Cloud

```yaml
providersConfiguration:
  alicloud:
    machines:
      general-2: "general-2 (2vCPU, 8GB RAM)"
      general-64: "general-64 (64vCPU, 256GB RAM)"

    machinesMapping:
      general-2: ecs.g9i.large
      general-64: ecs.g9i.16xlarge
```

## Resolution Logic

Instance types are resolved using a direct mapping between logical machine types and instance types.

The resolution process is:
1. Read the configured logical machine type (e.g., `general-2`).
2. Look up the corresponding instance type in `machinesMapping`.
3. Return the mapped instance type.

This approach completely decouples the logical machine definition from the instance family, making upgrades or replacements easier by modifying only the mapping.

# Comparison

| Aspect               | Semi-Abstract Configuration                      | Abstract Configuration                                                                                            |
|----------------------|--------------------------------------------------|-------------------------------------------------------------------------------------------------------------------|
| Configuration Length | Shorter configuration                            | Longer configuration                                                                                              |
| Flexibility          | Only the machine **version** can be updated      | The **name is fully abstract** and can be switched to a completely different machine type                         |
| Logic Complexity     | More complicated logic, varies for each provider | Straightforward logic that is the **same across all providers**                                                   |
| Potential Issues     | –                                                | **AWS:** two general types and two GPU types<br>**Azure:** two general machine types<br>Naming can be challenging |

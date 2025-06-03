# Regions Supporting Machine

## Overview

The **regionsSupportingMachine** configuration field defines machine type families that are not universally available across all regions. 
This configuration ensures that if a machine type family is listed, it is restricted to the explicitly specified regions, and optionally to specific zones within those regions.

If a region is listed without zones, the machine type is supported in all zones of that region.
When a new worker node pool is created, it uses the same zones as the Kyma worker node pool. If high availability (HA) is turned off, it uses just one of those zones.

If zones are specified, the machine type is only available in those zones within the region.
When a new worker node pool is created, three zones are randomly selected from the list provided in the configuration. If HA is disabled, only one of those zones is used.

| **Machine Type** |    **Region**    | **Specified Zones** | **Kyma Zones** | **HA** |                          **Provisioning Details**                          |
|:----------------:|:----------------:|:-------------------:|:--------------:|:------:|:---------------------------------------------------------------------------|
|      `m8g`       |  `ca-central-1`  |         `-`         |  `[a, b, c]`   |  true  |            Worker node pool provisioned in zones `a`, `b`, `c`             |
|      `m8g`       |  `ca-central-1`  |         `-`         |  `[a, b, c]`   | false  |   Worker node pool provisioned in a single random zone, for example, `a`   |
|   `Standard_L`   |   `japaneast`    |   `[a, b, c, d]`    |  `[a, b, c]`   |  true  | Worker node pool provisioned in 3 random zones, for example, `a`, `b`, `d` |
|   `Standard_L`   |   `japaneast`    |   `[a, b, c, d]`    |  `[a, b, c]`   | false  |   Worker node pool provisioned in a single random zone, for example, `d`   |
|      `m8g`       | `ap-northeast-1` |      `[a, b]`       |  `[a, b, c]`   |  true  |     Error message saying that machine type is not available in 3 zones     |
|      `m8g`       | `ap-northeast-1` |      `[a, b]`       |  `[a, b, c]`   | false  |   Worker node pool provisioned in a single random zone, for example, `a`   |

See a sample configuration:

```yaml
regionsSupportingMachine: |-
  m8g:
    ap-northeast-1: [a, b]
    ap-southeast-1:
    ca-central-1:
  c2d-highmem:
    us-central1:
    southamerica-east1:
  Standard_L:
    uksouth:
    japaneast: [a, b, c, d]
    brazilsouth:
```

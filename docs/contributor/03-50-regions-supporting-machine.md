# Regions Supporting Machine

## Overview

The **regionsSupportingMachine** configuration field defines machine type families that are not universally available across all regions. 
This configuration ensures that if a machine type family is listed, it is restricted to the explicitly specified regions, and optionally to specific zones within those regions.
If a region is listed without zones, the machine type is supported in all zones of that region. If zones are specified, the machine type is only available in those zones within the region.

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
    japaneast: [a, b, c]
    brazilsouth:
```

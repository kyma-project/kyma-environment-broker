# Regions Supporting Machine

## Overview

The **regionsSupportingMachine** configuration field defines machine type families that are not universally available across all regions. 
This configuration ensures that if a machine type family is listed, it is restricted to the explicitly specified regions.

See a sample configuration:

```yaml
regionsSupportingMachine: |-
  m8g:
    - ap-northeast-1
    - ap-southeast-1
    - ca-central-1
  c2d-highmem:
    - us-central1
    - southamerica-east1
  Standard_L:
    - uksouth
    - japaneast
    - brazilsouth
```

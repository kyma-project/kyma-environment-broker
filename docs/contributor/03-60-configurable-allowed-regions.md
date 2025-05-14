## Regions and Zones Configuration

Most of the Kyma plans requires a region to be specified in the request body (provisioning parameters). The regions configuration  allows you to control which regions are available for the Kyma plan and a platform region. Additionally, every region has different zones which needs to be specified. The configuration is defined in the `values.yaml` file.

# Allowed regions

The `plansConfiguration` section of the `values.yaml` file contains a list of plans. Every plan has a mapping - platform region to allowed provider regions. To avoid specifying all possible platform regions you can define `default` values, for example:

```yaml

plansConfiguration:
  aws:
    cf-eu11:
      - eu-central-1
    default:
      - us-east-1
      - us-west-2
```

The above configuration means that the `cf-eu11` platform region is allowed to use only the `eu-central-1` provider region, while all other platform regions are allowed to use `us-east-1` and `us-west-2` provider regions.

# Display names and zones

The json schema, which defines allowed regions, contains a list of a region display name, which is shown to the user. The mapping between the region code and the display name is defined in the `providersCofiguration`. Mappings are grouped by a provider. Every region has two properties: displayName and zones. The display name is shown to the user in the UI, while the zones are used by the provisioning process to create a worker node pool.
```yaml
providersConfiguration:
  aws:
    regions:
      eu-central-1:
          displayName: "eu-central-1 (Europe, Frankfurt)"
          zones: [ "a", "b", "c" ]
      eu-west-2:
        displayName: "eu-west-2 (Europe, London)"
        zones: [ "a", "b", "c" ]
  azure:
    regions:
      brazilsouth: 
        displayName: "Brazil South"
        zones: [ "1", "2", "3" ]
      canadacentral: 
        displayName: "Canada Central"
        zones: [ "1", "2", "3" ]
```

The above configuration defines display names and zones for AWS and Azure regions. You can describe regions for the following providers: azure, aws, gcp, sap-converged-cloud.

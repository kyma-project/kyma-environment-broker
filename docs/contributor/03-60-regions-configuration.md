# Regions and Zones Configuration

Most Kyma plans require specifying a region in the request body as one of the provisioning parameters. The regions' configuration allows you to control which regions are available for the Kyma plan and the platform region. Additionally, every region has a different set of zones.
The configuration is used to validate the region specified in the request body, and those regions are also used to generate schemas for the catalog endpoint.

## Allowed Regions

The `plansConfiguration` section of the `values.yaml` file contains a list of plans. Every plan has a platform region with allowed regions. To avoid specifying all possible platform regions, define `default` values, for example:

```yaml

plansConfiguration:
  aws:
    cf-eu11:
      - eu-central-1
    default:
      - us-east-1
      - us-west-2
```

The above configuration means that the `cf-eu11` platform region can only use the `eu-central-1` provider region, while all other platform regions can use the `us-east-1` and `us-west-2` provider regions. 
The trial and free plans are not listed in the `plansConfiguration` section. The list of allowed regions for the free plan is calculated in the code. The trial plan does not support the **region** parameter.

> ### Note:
> If your configuration does not contain any region for a given plan, this plan is not present in the catalog endpoint and cannot be used for provisioning.


## Display Names and Zones

The JSON schema, which defines allowed regions, contains a list of region display names, which are shown to the user. Each region must have a corresponding display name and a set of zones defined in the `providersConfiguration` section. The display name is shown to the user in the UI, while the zones are used by the provisioning process to create a worker node pool.
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

The above configuration defines display names and zones for AWS and Azure regions. You can describe regions for the following providers: `azure`, `aws`, `gcp`, `sap-converged-cloud`.
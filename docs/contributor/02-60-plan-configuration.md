# Plan Configuration

According to the Open Service Broker API (OSB API) specification, KEB supports Kyma's multiple plans. Each plan has its own configuration, which specifies allowed regions, zones, machine types, and their display names. This document describes an overview of the plan configuration.

## Enabling plan

The `enablePlans` property contains a comma-separated supported plan names. To enable a plan, add the name to the list, for example:
```yaml
enablePlans: "trial,aws,gcp"
```

## HAP Rules

Each Kyma needs a subscription for the hyperscaler. The HAP Rule configuration allows you to define a rule how the subscription is selected, for example:

```yaml
hap:
  rule:
    - aws(PR=cf-eu11) -> EU
    - aws
```

Every plan must have at least one HAP rule defined.
You can find more details in the [Hyperscaler Account Pool Rules](03-11-hap-rules.md) document.

## Configure Plan and Provider Details

Every plan has its own configuration, which allows you to specify details of a plan. You can specify more than one plan as a key if the configuration is the same, for example:

```yaml
plansConfiguration:
  
  # one or more plans can be defined
  aws,build-runtime-aws:
    
      # defines allowed plan changes
      upgradableToPlans:
        - build-runtime-aws
      
      # volume size in GB
      volumeSizeGb: 80
      
      # defines a list of machine types
      regularMachines:
        - "m6i.large"
        - "m6i.xlarge"
      
      # defines additional machines, which can be used only in additional worker node pools
      additionalMachines:
        - "c7i.large"
        - "c7i.xlarge"
      
      # defines a list of regions where the plan can be used grouped by BTP region
      regions:
        cf-eu11:
          - "eu-central-1"
        default:
          - "eu-central-1"
          - "eu-west-2"
  
```

Each provider has its own configuration which defines provider details, for example:

```yaml
providersConfiguration:
  aws:
    # enables dual-stack networking support (IPv4 and IPv6)
    dualStack: true
    
    # machine display names
    machines:
      "m6i.large": "m6i.large (2vCPU, 8GB RAM)"
      "m6i.xlarge": "m6i.xlarge (4vCPU, 16GB RAM)"
      
    # machine type families that are not universally available across all regions
    regionsSupportingMachine:
      g6:
        eu-central-1: [a, b]
        
    # region display names and zones
    regions:
      eu-central-1:
          displayName: "eu-central-1 (Europe, Frankfurt)"
          zones: ["a", "b", "c"]
      eu-west-2:
          displayName: "eu-west-2 (Europe, London)"
          zones: ["a", "b", "c"]

    # defines whether Kyma Environment Broker determines availability zones dynamically from the hyperscaler
    # or uses the static zones defined in the provider configuration
    zonesDiscovery: false
```
You can find more details in the following documents:
 * [Regions Configuration](03-60-regions-configuration.md)
 * [Machine Types Configuration](03-70-machines-configuration.md)
 * [Regions Supporting Machine Types](03-50-regions-supporting-machine.md)
 * [Zones Discovery](03-55-zones-discovery.md)
 * [Plan Updates](03-80-plan-updates.md)
 * [Dual-Stack Configuration](03-85-dual-stack-configuration.md)

## Bindings

Bindings allows generating credentials for accessing the cluster. To enable bindigns for a given plan, you must add a plan name to the `bindablePlans` list in the `broker.binding` section of the configuration. For example, to enable bindings for the `aws` plan, you can use the following configuration:

```yaml
broker:
  binding:
    bindablePlans: aws
```

> [!NOTE]
> Bindings are not required to create a Kyma instance.

For more information, see [Kyma Bindings](../user/05-60-kyma-bindings.md).

## Kyma Custom Resource Template Configuration

Kyma Environment Broker (KEB) uses the Kyma custom resource template to create a Kyma CR. If you want to define a custom Kyma CR template, define the `runtimeConfiguration` setting according to [Kyma Custom Resource Template Configuration](02-40-kyma-template.md), for example:

````yaml
runtimeConfiguration: |-
  default: |-
    kyma-template: |-
      apiVersion: operator.kyma-project.io/v1beta2
      kind: Kyma
      metadata:
        labels:
          "operator.kyma-project.io/managed-by": "lifecycle-manager"
        name: tbd
        namespace: kcp-system
      spec:
        channel: regular
        modules: []
````

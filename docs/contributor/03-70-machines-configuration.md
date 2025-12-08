# Machine Types Configuration

Most Kyma plans require specifying a machine type in the request body as one of the provisioning parameters. The machine types configuration allows you to control which machine types are available for a Kyma plan and how they are displayed in the UI.

## Allowed Machine Types

The `plansConfiguration` property contains a list of plans. Every plan has a machine types configuration. Each plan has a **regularMachines** list containing a list of available values for the **machineType** parameter. The **additionalMachines** list contains machines which are available only in additional worker node pools, for example:

```yaml
plansConfiguration:
  aws:
    regularMachines:
        - Standard_D2s_v5
        - Standard_D4s_v5
    additionalMachines:
        - Standard_D8s_v5
        - Standard_D16s_v5
```

The above configuration means that the `Standard_D2s_v5` and `Standard_D4s_v5` machine types are available for the **machineType** parameter. The additional worker node pool can use the following machine types: `Standard_D2s_v5`, `Standard_D4s_v5`, `Standard_D8s_v5`, `Standard_D16s_v5`.

> ### Note:
> The free and trial plans do not support the **machineType** parameter. The machine type is hardcoded for these plans.

## Display Names

The catalog endpoint provides display names for machine types. The display names are defined in the `providersConfiguration` section of the `values.yaml` file, for example:

```yaml
providersConfiguration:
  aws:
    machineTypes:
      Standard_D2s_v5: "Standard D2s v5 (2 vCPUs, 8 GiB RAM)"
      Standard_D4s_v5: "Standard D4s v5 (4 vCPUs, 16 GiB RAM)"
      Standard_D8s_v5: "Standard D8s v5 (8 vCPUs, 32 GiB RAM)"
      Standard_D16s_v5: "Standard D16s v5 (16 vCPUs, 64 GiB RAM)"
```

# Adding a New Machine Type to a Plan

As an operator, you add and manage machine types available in a given plan. When adding a new machine type, follow the procedure described in this document to avoid provisioning failures.

> ### Note:
> When a customer requests a new machine type in the production environment, handle the request directly. If the request concerns a canary environment (ns2-canary, china-canary), forward it to the Kyma team and stay involved in the process.

## Check Zone Availability

For every region listed under the plan's **plansConfiguration.\<plan\>.regions**, verify that the new machine type is available in that region and in a sufficient number of its zones.

Machine types are not universally available across all regions and zones. If a machine type is only available in specific regions or zones, it must be listed under **providersConfiguration.\<provider\>.regionsSupportingMachine** instead of being added as a general machine.

The Kyma worker node pool always runs in high availability (HA) mode and requires at least three zones in every supported region. Only add a machine type to **regularMachines** if it is available in at least three zones across all regions supported by the plan. If this condition is not met, KEB rejects provisioning. If zones are specified in **regionsSupportingMachine**, three zones are randomly selected from the provided list. If only one or two zones are available, HA provisioning fails with an error.

Additional worker node pools can ru in both HA and non-HA modes. With HA disabled, one zone is sufficient. If zones are specified, one is randomly selected from the list.

If a region is listed without zones, the machine type is considered available in all zones of that region and zone selection follows the zones assigned to the Kyma worker node pool in the same region.

For example:

```yaml
providersConfiguration:
  aws:
    regionsSupportingMachine:
      g6:
        eu-central-1: [a, b, c]
        ap-south-1: [b]
```

To verify availability, check the hyperscaler's documentation for the specific machine type in each supported region.

For more information, see [Regions Supporting Machine Types](03-50-regions-supporting-machine.md) and [Zones Discovery](03-55-zones-discovery.md).

## Configuration Steps

Once the zone availability check is complete, proceed with the configuration:

1. Add the machine type to **plansConfiguration.\<plan\>.regularMachines** (for use as the main worker node pool machine type) or **additionalMachines** (for use in additional worker node pools only). The first entry in **regularMachines** becomes the default machine type for the plan.

2. Add the display name to **providersConfiguration.\<provider\>.machines**.

3. If the machine type is not available in all regions or zones supported by the plan, add the appropriate entry to **providersConfiguration.\<provider\>.regionsSupportingMachine**.

For full configuration reference, see [Machine Types Configuration](03-70-machines-configuration.md), [Regions Supporting Machine Types](03-50-regions-supporting-machine.md), and [Plan Configuration](02-60-plan-configuration.md).

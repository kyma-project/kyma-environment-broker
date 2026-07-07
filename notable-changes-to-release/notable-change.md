<!--{"metadata":{"requirement":"MANDATORY","type":"EXTERNAL","category":"FEATURE"}}-->

# KEB: Labels and Annotations for Additional Worker Node Pools Enabled by Default

> ### Caution:
> This update is mandatory. Without performing it, you will not be able to use the feature in the SAP BTP cockpit.

## What's Changed

Labels and annotations on additional worker node pools are now enabled.

## Procedure

Refresh the ERS schema to include the `labels` and `annotations` fields in the additional worker node pool configuration.

## Post-Update Steps

Verify that in the SAP BTP cockpit, the **Labels** and **Annotations** fields are visible in the configuration window under the **Additional Worker Node Pools** section.
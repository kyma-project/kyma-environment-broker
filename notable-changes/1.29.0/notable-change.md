<!--{"metadata":{"requirement":"MANDATORY","type":"EXTERNAL","category":"FEATURE"}}-->

# KEB: gVisor Container Runtime for the Main Worker Pool and Additional Worker Node Pools

> ### Caution:
> This update is mandatory. Without performing it, you will not be able to use the feature in the SAP BTP cockpit.

## Prerequisites

Access to Environment Registry Service (ERS).

## What's Changed

For cloud-native container security and portability, you can enable the **gVisor** container runtime on the main Kyma worker pool and on individual additional worker node pools. gVisor provides an additional layer of isolation between containers and the host kernel. The feature is gated behind a global account whitelist.

## Procedure

Refresh broker details using the xRS APIs in ERS.

## Post-Update Steps

Verify that in the SAP BTP cockpit, the **Gvisor** field is visible in the configuration window for whitelisted global accounts.

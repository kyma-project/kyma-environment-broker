# EU Access

## Overview

EU Access requires, among others, that Data Residency is in the European Economic Area or Switzerland. 

SAP BTP, Kyma runtime supports the BTP `cf-eu11` AWS and BTP `cf-ch20` Azure subaccount regions, which are
called EU Access BTP subaccount regions. 
Kyma Control Plane manages `cf-eu11` Kyma runtimes in a separate AWS hyperscaler account pool and 
`cf-ch20` Kyma runtimes in a separate Azure hyperscaler account pool.

When the PlatformRegion is an EU access BTP subaccount region:
- Kyma Environment Broker (KEB) provides the **euAccess** parameter to Provisioner
- KEB services catalog handler exposes:
  - `eu-central-1` as the only possible value for the **region** parameter for `cf-eu11` 
  - `switzerlandnorth` as the only possible value for the **region** parameter for `cf-ch20`

## Access 
Due to limited availability, the provisioning request for the EU Access only regions can succeed only if GlobalAccountId 
is added to the list of allowed GlobalAccountIds (the whitelist).
Before attempting to provision Kyma clusters in the EU Access only regions, you must open a support ticket to have your 
GlobalAccountId added to the whitelist.

If the GlobalAccountId for the provisioning request is not whitelisted, KEB responds 
with `http code 400` (Bad Request) and the message preconfigured in `management-plane-config`. 
The user gets the following message in the SAP BTP cockpit.   
```yaml
kcp-prod:
  kyma-environment-broker:
    euAccessRejectionMessage: "Due to limited availability, you must open a support ticket before attempting to provision Kyma clusters in the EU Access only regions"
```

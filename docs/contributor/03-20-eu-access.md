# EU Access

## Overview

EU Access requires, among others, that Data Residency is in the European Economic Area or Switzerland.

SAP BTP, Kyma runtime supports the BTP `cf-eu11` AWS and BTP `cf-ch20` Azure subaccount regions, which are
called EU Access BTP subaccount regions.
Kyma Control Plane manages `cf-eu11` Kyma runtimes in a separate AWS hyperscaler account pool and
`cf-ch20` Kyma runtimes in a separate Azure hyperscaler account pool.

When the PlatformRegion is an EU access BTP subaccount region:

* Kyma Environment Broker (KEB) provides the **euAccess** parameter to Kyma Infrastructure Manager
* KEB services catalog handler exposes:
   * `eu-central-1` as the only possible value for the **region** parameter for `cf-eu11`
   * `switzerlandnorth` as the only possible value for the **region** parameter for `cf-ch20`

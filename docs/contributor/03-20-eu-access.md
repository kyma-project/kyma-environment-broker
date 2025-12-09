# EU Access

EU Access ensures that your data residency is in the European Economic Area or Switzerland.

SAP BTP, Kyma runtime supports the following EU Access BTP subaccount regions:
- `cf-eu11` (AWS)
- `cf-ch20` (Azure)
- `cf-eu01` (SAP Cloud Infrastructure)
- `cf-eu02` (SAP Cloud Infrastructure)

When the **PlatformRegion** is an EU access BTP subaccount region, the following happens:
- Kyma Environment Broker (KEB) provides the **euAccess** parameter to Kyma Infrastructure Manager (KIM)
- KEB service catalog restricts the available **region** parameter to the following supported cluster regions:

  | BTP Subaccount Region |   Cluster Region   |
  |:---------------------:|:------------------:|
  |       `cf-eu11`       |   `eu-central-1`   |
  |       `cf-ch20`       | `switzerlandnorth` |
  |       `cf-eu01`       |     `eu-de-2`      |
  |       `cf-eu02`       |     `eu-de-1`      |

See examples of Kyma Control Plane-managed EU access configurations.

- The `cf-eu11` Kyma runtimes using a dedicated AWS hyperscaler account pool with EU Access enabled:
  ```yaml
  hap:
    rule:
      - aws(PR=cf-eu11) -> EU # pool: hyperscalerType: aws; euAccess: true
  ```

- The `cf-ch20` Kyma runtimes using a dedicated Azure hyperscaler account pool with EU Access enabled:
  ```yaml
  hap:
    rule:
      - azure(PR=cf-ch20) -> EU # pool: hyperscalerType: azure; euAccess: true
  ```

- The `cf-eu01` and `cf-eu02` Kyma runtimes using dedicated SAP Cloud Infrastructure account pools:
  ```yaml
  hap:
    rule:
      - sap-converged-cloud -> S, HR # pool: hyperscalerType: openstack_<HYPERSCALER_REGION>; shared: true
  ```

  For these BTP subaccount regions, EU Access is not explicitly enabled because each hyperscaler region uses a separate account pool. 
  Since `eu-de-1` is only available in `cf-eu02` and `eu-de-2` only in `cf-eu01`, all clusters in these regions inherently comply with EU Access requirements.

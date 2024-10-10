# Service Description

Kyma Environment Broker (KEB) is compatible with the [Open Service Broker API (OSBAPI)](https://www.openservicebrokerapi.org/) specification. It provides a ServiceClass that provisions SAP BTP, Kyma runtime in a cluster.

## Service Plans

The supported plans are as follows:

| Plan name             | Plan ID                                | Description                                             |
|-----------------------|----------------------------------------|---------------------------------------------------------|
| `azure`               | `4deee563-e5ec-4731-b9b1-53b42d855f0c` | Installs Kyma runtime in the Azure cluster.             |
| `azure_lite`          | `8cb22518-aa26-44c5-91a0-e669ec9bf443` | Installs Kyma Lite in the Azure cluster.                |
| `aws`                 | `361c511f-f939-4621-b228-d0fb79a1fe15` | Installs Kyma runtime in the AWS cluster.               |
| `gcp`                 | `ca6e5357-707f-4565-bbbd-b3ab732597c6` | Installs Kyma runtime in the GCP cluster.               |
| `trial`               | `7d55d31d-35ae-4438-bf13-6ffdfa107d9f` | Installs Kyma trial plan on Azure, AWS or GCP.          |
| `sap-converged-cloud` | `03b812ac-c991-4528-b5bd-08b303523a63` | Installs Kyma runtime in the SapConvergedCloud cluster. |
| `free`                | `b1a5764e-2ea1-4f95-94c0-2b4538b37b55` | Installs Kyma free plan on Azure or AWS.                |

There is also an experimental plan:

| Plan name | Plan ID                                | Description                                           |
|-----------|----------------------------------------|-------------------------------------------------------|
| `preview` | `5cb3d976-b85c-42ea-a636-79cadda109a9` | Installs Kyma runtime on AWS using Lifecycle Manager. |

> [!WARNING] 
> The experimental plan may fail to work or be removed.

## Provisioning Parameters

There are two types of configurable provisioning parameters: the ones that are compliant for all providers and provider-specific ones.

### Parameters Compliant for All Providers

These are the provisioning parameters that you can configure:

| Parameter name                                   | Type   | Description                                                                                                      | Required | Default value   |
|--------------------------------------------------|--------|------------------------------------------------------------------------------------------------------------------|:--------:|-----------------|
| **name**                                         | string | Specifies the name of the cluster.                                                                               |   Yes    | None            |
| **purpose**                                      | string | Provides a purpose for a Kyma runtime.                                                                           |    No    | None            |
| **targetSecret**                                 | string | Provides the name of the Secret that contains hyperscaler's credentials for a Kyma runtime.                      |    No    | None            |
| **platform_region**                              | string | Defines the platform region that is sent in the request path.                                                    |    No    | None            |
| **platform_provider**                            | string | Defines the platform provider for a Kyma runtime.                                                                |    No    | None            |
| **context.tenant_id**                            | string | Provides a tenant ID for a Kyma runtime.                                                                         |    No    | None            |
| **context.subaccount_id**                        | string | Provides a subaccount ID for a Kyma runtime.                                                                     |    No    | None            |
| **context.globalaccount_id**                     | string | Provides a global account ID for a Kyma runtime.                                                                 |    No    | None            |
| **context.sm_operator_credentials.clientid**     | string | Provides a client ID for SAP BTP service operator.                                                               |    No    | None            |
| **context.sm_operator_credentials.clientsecret** | string | Provides a client secret for the SAP BTP service operator.                                                       |    No    | None            |
| **context.sm_operator_credentials.sm_url**       | string | Provides a SAP Service Manager URL for the SAP BTP service operator.                                             |    No    | None            |
| **context.sm_operator_credentials.url**          | string | Provides an authentication URL for the SAP BTP service operator.                                                 |    No    | None            |
| **context.sm_operator_credentials.xsappname**    | string | Provides an XSApp name for the SAP BTP service operator.                                                         |    No    | None            |
| **context.user_id**                              | string | Provides a user ID for a Kyma runtime.                                                                           |    No    | None            |
| **oidc.clientID**                                | string | Provides an OIDC client ID for a Kyma runtime.                                                                   |    No    | None            |
| **oidc.groupsClaim**                             | string | Provides an OIDC groups claim for a Kyma runtime.                                                                |    No    | `groups`        |
| **oidc.issuerURL**                               | string | Provides an OIDC issuer URL for a Kyma runtime.                                                                  |    No    | None            |
| **oidc.signingAlgs**                             | string | Provides the OIDC signing algorithms for a Kyma runtime.                                                         |    No    | `RS256`         |
| **oidc.usernameClaim**                           | string | Provides an OIDC username claim for a Kyma runtime.                                                              |    No    | `email`         |
| **oidc.usernamePrefix**                          | string | Provides an OIDC username prefix for a Kyma runtime.                                                             |    No    | None            |
| **administrators**                               | string | Provides administrators for a Kyma runtime.                                                                      |    No    | None            |
| **networking.nodes**                             | string | The Node network's CIDR.                                                                                         |    No    | `10.250.0.0/22` |
| **modules.default**                              | bool   | Defines whether to use a default list of modules                                                                 |    No    | None            |
| **modules.list**                                 | array  | Defines a custom list of modules                                                                                 |    No    | None            |

### Provider-specific Parameters

These are the provisioning parameters for Azure that you can configure:

<div tabs name="azure-plans" group="azure-plans">
  <details>
  <summary label="azure-plan">
  Azure
  </summary>

<!-- markdown-link-check-disable -->

| Parameter name                            | Type   | Description                                                                         | Required | Default value     |
|-------------------------------------------|--------|-------------------------------------------------------------------------------------|:--------:|-------------------|
| **machineType**                           | string | Specifies the provider-specific virtual machine type.                               |    No    | `Standard_D2s_v5` |
| **volumeSizeGb**                          | int    | Specifies the size of the root volume.                                              |    No    | `50`              |
| **region**                                | string | Defines the cluster region.                                                         |   Yes    | None              |
| **zones**                                 | string | Defines the list of zones in which Runtime Provisioner creates a cluster.           |    No    | `["1"]`           |
| **autoScalerMin[<sup>1</sup>](#update)**  | int    | Specifies the minimum number of virtual machines to create.                         |    No    | `2`               |
| **autoScalerMax[<sup>1</sup>](#update)**  | int    | Specifies the maximum number of virtual machines to create, up to `40` allowed.     |    No    | `10`              |
| **maxSurge[<sup>1</sup>](#update)**       | int    | Specifies the maximum number of virtual machines that are created during an update. |    No    | `4`               |
| **maxUnavailable[<sup>1</sup>](#update)** | int    | Specifies the maximum number of VMs that can be unavailable during an update.       |    No    | `1`               |

<!-- markdown-link-check-enable-->

  </details>
  <details>
  <summary label="azure-lite-plan">
  Azure Lite
  </summary>

<!-- markdown-link-check-disable -->

| Parameter name                            | Type   | Description                                                                         | Required | Default value     |
|-------------------------------------------|--------|-------------------------------------------------------------------------------------|:--------:|-------------------|
| **machineType**                           | string | Specifies the provider-specific virtual machine type.                               |    No    | `Standard_D4s_v5` |
| **volumeSizeGb**                          | int    | Specifies the size of the root volume.                                              |    No    | `50`              |
| **region**                                | string | Defines the cluster region.                                                         |   Yes    | None              |
| **zones**                                 | string | Defines the list of zones in which Runtime Provisioner creates a cluster.           |    No    | `["1"]`           |
| **autoScalerMin[<sup>1</sup>](#update)**  | int    | Specifies the minimum number of virtual machines to create.                         |    No    | `2`               |
| **autoScalerMax[<sup>1</sup>](#update)**  | int    | Specifies the maximum number of virtual machines to create, up to `40` allowed.     |    No    | `10`              |
| **maxSurge[<sup>1</sup>](#update)**       | int    | Specifies the maximum number of virtual machines that are created during an update. |    No    | `4`               |
| **maxUnavailable[<sup>1</sup>](#update)** | int    | Specifies the maximum number of VMs that can be unavailable during an update.       |    No    | `1`               |

<!-- markdown-link-check-enable-->

 </details>
 </div>

These are the provisioning parameters for AWS that you can configure:
<div tabs name="aws-plans" group="aws-plans">
  <details>
  <summary label="aws-plan">
  AWS
  </summary>

<!-- markdown-link-check-disable -->

| Parameter name                            | Type   | Description                                                                                | Required | Default value |
|-------------------------------------------|--------|--------------------------------------------------------------------------------------------|:--------:|---------------|
| **machineType**                           | string | Specifies the provider-specific virtual machine type.                                      |    No    | `m6i.large`   |
| **volumeSizeGb**                          | int    | Specifies the size of the root volume.                                                     |    No    | `50`          |
| **region**                                | string | Defines the cluster region.                                                                |   Yes    | None          |
| **zones**                                 | string | Defines the list of zones in which Runtime Provisioner creates a cluster.                  |    No    | `["1"]`       |
| **autoScalerMin[<sup>1</sup>](#update)**  | int    | Specifies the minimum number of virtual machines to create.                                |    No    | `3`           |
| **autoScalerMax[<sup>1</sup>](#update)**  | int    | Specifies the maximum number of virtual machines to create, up to `40` allowed.            |    No    | `10`          |
| **maxSurge[<sup>1</sup>](#update)**       | int    | Specifies the maximum number of virtual machines that are created during an update.        |    No    | `4`           |
| **maxUnavailable[<sup>1</sup>](#update)** | int    | Specifies the maximum number of virtual machines that can be unavailable during an update. |    No    | `1`           |

<!-- markdown-link-check-enable-->

  </details>
 </div>

These are the provisioning parameters for GCP that you can configure:

<div tabs name="gcp-plans" group="gcp-plans">
  <details>
  <summary label="gcp-plan">
  GCP
  </summary>

<!-- markdown-link-check-disable -->

| Parameter name                            | Type   | Description                                                                         | Required | Default value   |
|-------------------------------------------|--------|-------------------------------------------------------------------------------------|:--------:|-----------------|
| **machineType**                           | string | Specifies the provider-specific virtual machine type.                               |    No    | `n2-standard-2` |
| **volumeSizeGb**                          | int    | Specifies the size of the root volume.                                              |    No    | `30`            |
| **region**                                | string | Defines the cluster region.                                                         |   Yes    | None            |
| **zones**                                 | string | Defines the list of zones in which Runtime Provisioner creates a cluster.           |    No    | `["a"]`         |
| **autoScalerMin[<sup>1</sup>](#update)**  | int    | Specifies the minimum number of virtual machines to create.                         |    No    | `3`             |
| **autoScalerMax[<sup>1</sup>](#update)**  | int    | Specifies the maximum number of virtual machines to create.                         |    No    | `4`             |
| **maxSurge[<sup>1</sup>](#update)**       | int    | Specifies the maximum number of virtual machines that are created during an update. |    No    | `4`             |
| **maxUnavailable[<sup>1</sup>](#update)** | int    | Specifies the maximum number of VMs that can be unavailable during an update.       |    No    | `1`             |

<!-- markdown-link-check-enable -->

 </details>
 </div>

These are the provisioning parameters for SapConvergedCloud that you can configure:

<div tabs name="sap-converged-cloud-plans" group="sap-converged-cloud-plans">
  <details>
  <summary label="sap-converged-cloud-plan">
  SapConvergedCloud
  </summary>

<!-- markdown-link-check-disable -->

| Parameter name                            | Type   | Description                                                                                | Required | Default value |
|-------------------------------------------|--------|--------------------------------------------------------------------------------------------|:--------:|---------------|
| **machineType**                           | string | Specifies the provider-specific virtual machine type.                                      |    No    | `g_c2_m8`     |
| **volumeSizeGb**                          | int    | Specifies the size of the root volume.                                                     |    No    | `30`          |
| **region**                                | string | Defines the cluster region.                                                                |   Yes    | None          |
| **zones**                                 | string | Defines the list of zones in which Runtime Provisioner creates a cluster.                  |    No    | `["a"]`       |
| **autoScalerMin[<sup>1</sup>](#update)**  | int    | Specifies the minimum number of virtual machines to create.                                |    No    | `3`           |
| **autoScalerMax[<sup>1</sup>](#update)**  | int    | Specifies the maximum number of virtual machines to create.                                |    No    | `20`          |
| **maxSurge[<sup>1</sup>](#update)**       | int    | Specifies the maximum number of virtual machines that are created during an update.        |    No    | `4`           |
| **maxUnavailable[<sup>1</sup>](#update)** | int    | Specifies the maximum number of virtual machines that can be unavailable during an update. |    No    | `1`           |

<!-- markdown-link-check-enable -->

The SAP Converged Cloud plan cannot be provisioned in all SKR regions. This restriction is enforced through the region mapping functionality configured by [`sapConvergedCloudPlanRegionMappings`](https://github.com/kyma-project/kyma-environment-broker/blob/48d5f55dfacfc511ead132fb77f522abc7e382e7/resources/keb/values.yaml#L215). The lists enable you to map a BTP region, which is passed to the provisioning endpoint in an HTTP path parameter (map key), to Kyma regions (list entries). Based on that configuration and the passed path parameter, the broker schema is populated only with values from the mapped list. In case of an empty mapping configuration or passing a provisioning path parameter that does not contain the configured region, the `sap-converged-cloud` plan is not rendered in the schema.

 </details>
 </div>

## Trial Plan

The trial plan allows you to install Kyma runtime on Azure, AWS, or GCP. The plan assumptions are as follows:
- Kyma runtime is uninstalled after 14 days and the Kyma cluster is deprovisioned after this time.
- It's possible to provision only one Kyma runtime per global account.

To reduce the costs, the trial plan skips one of the [provisioning steps](./03-20-runtime-operations.md#provisioning), that is, `AVS External Evaluation`.

### Provisioning Parameters

These are the provisioning parameters for the Trial plan that you can configure:

<div tabs name="trial-plan" group="trial-plan">
  <details>
  <summary label="trial-plan">
  Trial plan
  </summary>

| Parameter name     | Type   | Description                                                       | Required | Possible values       | Default value                       |
|--------------------|--------|-------------------------------------------------------------------|----------|-----------------------|-------------------------------------|
| **name**           | string | Specifies the name of the Kyma runtime.                           | Yes      | Any string            | None                                |
| **region**         | string | Defines the cluster region.                                       | No       | `europe`,`us`, `asia` | Calculated from the platform region |
| **provider**       | string | Specifies the cloud provider used during provisioning.            | No       | `Azure`, `AWS`, `GCP` | `Azure`                             |
| **context.active** | string | Specifies if the Kyma runtime should be suspended or unsuspended. | No       | `true`, `false`       | None                                |

The **region** parameter is optional. If not specified, the region is calculated from platform region specified in this path:
```shell
/oauth/{platform-region}/v2/service_instances/{instance_id}
```
The mapping between the platform region and the provider region (Azure, AWS or GCP) is defined in the configuration file in the **APP_TRIAL_REGION_MAPPING_FILE_PATH** environment variable. If the platform region is not defined, the default value is `europe`.

 </details>
 </div>

## Own Cluster Plan

> [!NOTE] 
> The `own_cluster` plan has been deprecated.

These are the provisioning parameters for the `own_cluster` plan that you configure:

<div tabs name="own_cluster-plan" group="own_cluster-plan">
  <details>
  <summary label="own_cluster-plan">
  Own cluster plan
  </summary>

| Parameter name  | Type   | Description                                                          | Required | Default value |
|-----------------|--------|----------------------------------------------------------------------|----------|---------------|
| **kubeconfig**  | string | Kubeconfig that points to the cluster where you instal Kyma runtime. | Yes      | None          |
| **shootDomain** | string | Domain of the shoot where you install Kyma runtime.                  | Yes      | None          |
| **shootName**   | string | Name of the shoot where you install Kyma runtime.                    | Yes      | None          |

</details>
</div>

## Preview Cluster Plan

The preview plan is designed for testing major changes in KEB's architecture.

### Provisioning Parameters

These are the provisioning parameters for the `preview` plan that you configure:

<div tabs name="preview_cluster-plan" group="preview_cluster-plan">
  <details>
  <summary label="preview_cluster-plan">
  Preview cluster plan
  </summary>

<!-- markdown-link-check-disable -->

| Parameter name                            | Type   | Description                                                                                | Required | Default value |
|-------------------------------------------|--------|--------------------------------------------------------------------------------------------|:--------:|---------------|
| **machineType**                           | string | Specifies the provider-specific virtual machine type.                                      |    No    | `m6i.large`   |
| **volumeSizeGb**                          | int    | Specifies the size of the root volume.                                                     |    No    | `50`          |
| **region**                                | string | Defines the cluster region.                                                                |   Yes    | None          |
| **zones**                                 | string | Defines the list of zones in which Runtime Provisioner creates a cluster.                  |    No    | `["1"]`       |
| **autoScalerMin[<sup>1</sup>](#update)**  | int    | Specifies the minimum number of virtual machines to create.                                |    No    | `3`           |
| **autoScalerMax[<sup>1</sup>](#update)**  | int    | Specifies the maximum number of virtual machines to create, up to `40` allowed.            |    No    | `10`          |
| **maxSurge[<sup>1</sup>](#update)**       | int    | Specifies the maximum number of virtual machines that are created during an update.        |    No    | `4`           |
| **maxUnavailable[<sup>1</sup>](#update)** | int    | Specifies the maximum number of virtual machines that can be unavailable during an update. |    No    | `1`           |

<!-- markdown-link-check-enable -->

</details>
</div>
<br>
<a name="update"><sup>1</sup> This parameter is available for <code>PATCH</code> as well, and can be updated with the same constraints as during provisioning.</a>

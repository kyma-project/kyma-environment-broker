# Service Description

Kyma Environment Broker (KEB) is compatible with the [Open Service Broker API (OSB API)](https://www.openservicebrokerapi.org/) specification. It provides a ServiceClass that provisions SAP BTP, Kyma runtime in a cluster.

## Service Plans

The supported plans that you can configure (see [Plan Configuration](../contributor/02-60-plan-configuration.md)) are as follows:

| Plan name                | Plan ID                                | Description                                                    |
|--------------------------|----------------------------------------|----------------------------------------------------------------|
| `azure`                  | `4deee563-e5ec-4731-b9b1-53b42d855f0c` | Installs Kyma runtime in the Azure cluster.                    |
| `azure_lite`             | `8cb22518-aa26-44c5-91a0-e669ec9bf443` | Installs Kyma Lite in the Azure cluster.                       |
| `aws`                    | `361c511f-f939-4621-b228-d0fb79a1fe15` | Installs Kyma runtime in the AWS cluster.                      |
| `gcp`                    | `ca6e5357-707f-4565-bbbd-b3ab732597c6` | Installs Kyma runtime in the Google Cloud cluster.             |
| `trial`                  | `7d55d31d-35ae-4438-bf13-6ffdfa107d9f` | Installs Kyma trial plan on Azure, AWS or Google Cloud.        |
| `sap-converged-cloud`    | `03b812ac-c991-4528-b5bd-08b303523a63` | Installs Kyma runtime in the SAP Cloud Infrastructure cluster. |
| `free`                   | `b1a5764e-2ea1-4f95-94c0-2b4538b37b55` | Installs Kyma free plan on Azure or AWS.                       |
| `build-runtime-aws`      | `6aae0ff3-89f7-4f12-86de-51466145422e` | Installs Kyma runtime in the AWS cluster.                      |
| `build-runtime-azure`    | `499244b4-1bef-48c9-be68-495269899f8e` | Installs Kyma runtime in the Azure cluster.                    |
| `build-runtime-gcp`      | `a310cd6b-6452-45a0-935d-d24ab53f9eba` | Installs Kyma runtime in the Google Cloud cluster.             |
| `alicloud`               | `9f2c3b4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d` | Installs Kyma runtime in the Alibaba Cloud cluster.            |
| `build-runtime-alicloud` | `72efa867-7b54-4d59-8df7-68f4759ff271` | Installs Kyma runtime in the Alibaba Cloud cluster.            |

There is also an experimental `preview` plan:

| Plan name | Plan ID                                | Description                                                |
|-----------|----------------------------------------|------------------------------------------------------------|
| `preview` | `5cb3d976-b85c-42ea-a636-79cadda109a9` | Installs Kyma runtime on AWS using Kyma Lifecycle Manager. |

> ### Caution:
> The experimental plan may fail to work or be removed.

## Provisioning Parameters

There are two types of configurable provisioning parameters: the ones that are compliant for all providers and provider-specific ones.

### Parameters Compliant for All Providers

You can configure the following provisioning parameters:

| Parameter name                                   | Type   | Description                                                                                 | Required | Default value   |
|--------------------------------------------------|--------|---------------------------------------------------------------------------------------------|:--------:|-----------------|
| **name**                                         | string | Specifies the name of the cluster.                                                          |   Yes    | None            |
| **purpose**                                      | string | Provides a purpose for a Kyma runtime.                                                      |    No    | None            |
| **targetSecret**                                 | string | Provides the name of the Secret that contains hyperscaler's credentials for a Kyma runtime. |    No    | None            |
| **platform_region**                              | string | Defines the platform region that is sent in the request path.                               |    No    | None            |
| **platform_provider**                            | string | Defines the platform provider for a Kyma runtime.                                           |    No    | None            |
| **context.tenant_id**                            | string | Provides a tenant ID for a Kyma runtime.                                                    |    No    | None            |
| **context.subaccount_id**                        | string | Provides a subaccount ID for a Kyma runtime.                                                |    No    | None            |
| **context.globalaccount_id**                     | string | Provides a global account ID for a Kyma runtime.                                            |    No    | None            |
| **context.sm_operator_credentials.clientid**     | string | Provides a client ID for SAP BTP service operator.                                          |    No    | None            |
| **context.sm_operator_credentials.clientsecret** | string | Provides a client Secret for the SAP BTP service operator.                                  |    No    | None            |
| **context.sm_operator_credentials.sm_url**       | string | Provides a SAP Service Manager URL for the SAP BTP service operator.                        |    No    | None            |
| **context.sm_operator_credentials.url**          | string | Provides an authentication URL for the SAP BTP service operator.                            |    No    | None            |
| **context.sm_operator_credentials.xsappname**    | string | Provides an XSApp name for the SAP BTP service operator.                                    |    No    | None            |
| **context.user_id**                              | string | Provides a user ID for a Kyma runtime.                                                      |    No    | None            |
| **oidc.clientID**                                | string | Provides an OIDC client ID for a Kyma runtime.                                              |    No    | None            |
| **oidc.groupsClaim**                             | string | Provides an OIDC groups claim for a Kyma runtime.                                           |    No    | `groups`        |
| **oidc.issuerURL**                               | string | Provides an OIDC issuer URL for a Kyma runtime.                                             |    No    | None            |
| **oidc.signingAlgs**                             | string | Provides the OIDC signing algorithms for a Kyma runtime.                                    |    No    | `RS256`         |
| **oidc.usernameClaim**                           | string | Provides an OIDC username claim for a Kyma runtime.                                         |    No    | `email`         |
| **oidc.usernamePrefix**                          | string | Provides an OIDC username prefix for a Kyma runtime.                                        |    No    | None            |
| **administrators**                               | string | Provides administrators for a Kyma runtime.                                                 |    No    | None            |
| **networking.nodes**                             | string | The node network's CIDR.                                                                    |    No    | `10.250.0.0/16` |
| **modules.channel**                              | string | Enables the user to define their preferred default release channel.                         |    No    | Taken from the runtimeConfiguration setting, where the Kyma resource spec channel is specified per plan. |
| **modules.default**                              | bool   | Defines whether to use a default list of modules.                                           |    No    | None            |
| **modules.list**                                 | array  | Defines a custom list of modules.                                                           |    No    | None            |

### Provider-Specific Parameters

You can configure the following provisioning parameters for Microsoft Azure:

<details>
<summary label="azure-plan">
Microsoft Azure
</summary>

| Parameter name                            | Type   | Description                                                                                   | Required | Default value     |
|-------------------------------------------|--------|-----------------------------------------------------------------------------------------------|:--------:|-------------------|
| **machineType**                           | string | Specifies the provider-specific virtual machine type.                                         |    No    | `Standard_D2s_v5` |
| **volumeSizeGb**                          | int    | Specifies the size of the root volume.                                                        |    No    | `50`              |
| **region**                                | string | Defines the cluster region.                                                                   |   Yes    | None              |
| **zones**                                 | string | Defines the list of zones in which Kyma Infrastructure Manager (KIM) creates a cluster.       |    No    | `["1"]`           |
| **autoScalerMin<sup>1</sup>**             | int    | Specifies the minimum number of virtual machines to create.                                   |    No    | `2`               |
| **autoScalerMax<sup>1</sup>**             | int    | Specifies the maximum number of virtual machines to create, up to `40` allowed.               |    No    | `10`              |
| **maxSurge<sup>1</sup>**                  | int    | Specifies the maximum number of virtual machines that are created during an update.           |    No    | `4`               |
| **maxUnavailable<sup>1</sup>**            | int    | Specifies the maximum number of virtual machines that can be unavailable during an update.    |    No    | `1`               |
| **additionalWorkerNodePools<sup>1</sup>** | array  | Defines a custom list of additional worker node pools.                                        |    No    | None              |

</details>

<details>
<summary label="azure-lite-plan">
Azure Lite
</summary>

| Parameter name                            | Type   | Description                                                                                   | Required | Default value     |
|-------------------------------------------|--------|-----------------------------------------------------------------------------------------------|:--------:|-------------------|
| **machineType**                           | string | Specifies the provider-specific virtual machine type.                                         |    No    | `Standard_D4s_v5` |
| **volumeSizeGb**                          | int    | Specifies the size of the root volume.                                                        |    No    | `50`              |
| **region**                                | string | Defines the cluster region.                                                                   |   Yes    | None              |
| **zones**                                 | string | Defines the list of zones in which KIM creates a cluster.                                     |    No    | `["1"]`           |
| **autoScalerMin<sup>1</sup>**             | int    | Specifies the minimum number of virtual machines to create.                                   |    No    | `2`               |
| **autoScalerMax<sup>1</sup>**             | int    | Specifies the maximum number of virtual machines to create, up to `40` allowed.               |    No    | `10`              |
| **maxSurge<sup>1</sup>**                  | int    | Specifies the maximum number of virtual machines that are created during an update.           |    No    | `4`               |
| **maxUnavailable<sup>1</sup>**            | int    | Specifies the maximum number of virtual machines that can be unavailable during an update.    |    No    | `1`               |
| **additionalWorkerNodePools<sup>1</sup>** | array  | Defines a custom list of additional worker node pools.                                        |    No    | None              |

</details>

You can configure the following provisioning parameters for Amazon Web Services (AWS):

<details>
<summary label="aws-plan">
AWS
</summary>

| Parameter name                            | Type   | Description                                                                                | Required | Default value |
|-------------------------------------------|--------|--------------------------------------------------------------------------------------------|:--------:|---------------|
| **machineType**                           | string | Specifies the provider-specific virtual machine type.                                      |    No    | `m6i.large`   |
| **volumeSizeGb**                          | int    | Specifies the size of the root volume.                                                     |    No    | `50`          |
| **region**                                | string | Defines the cluster region.                                                                |   Yes    | None          |
| **zones**                                 | string | Defines the list of zones in which KIM creates a cluster.                                  |    No    | `["1"]`       |
| **autoScalerMin<sup>1</sup>**             | int    | Specifies the minimum number of virtual machines to create.                                |    No    | `3`           |
| **autoScalerMax<sup>1</sup>**             | int    | Specifies the maximum number of virtual machines to create, up to `40` allowed.            |    No    | `10`          |
| **maxSurge<sup>1</sup>**                  | int    | Specifies the maximum number of virtual machines that are created during an update.        |    No    | `4`           |
| **maxUnavailable<sup>1</sup>**            | int    | Specifies the maximum number of virtual machines that can be unavailable during an update. |    No    | `1`           |
| **additionalWorkerNodePools<sup>1</sup>** | array  | Defines a custom list of additional worker node pools.                                     |    No    | None          |


</details>


You can configure the following provisioning parameters for Google Cloud:

<details>
<summary label="gcp-plan">
Google Cloud
</summary>

| Parameter name                            | Type   | Description                                                                                | Required | Default value   |
|-------------------------------------------|--------|--------------------------------------------------------------------------------------------|:--------:|-----------------|
| **machineType**                           | string | Specifies the provider-specific virtual machine type.                                      |    No    | `n2-standard-2` |
| **volumeSizeGb**                          | int    | Specifies the size of the root volume.                                                     |    No    | `30`            |
| **region**                                | string | Defines the cluster region.                                                                |   Yes    | None            |
| **zones**                                 | string | Defines the list of zones in which KIM creates a cluster.                                  |    No    | `["a"]`         |
| **autoScalerMin<sup>1</sup>**             | int    | Specifies the minimum number of virtual machines to create.                                |    No    | `3`             |
| **autoScalerMax<sup>1</sup>**             | int    | Specifies the maximum number of virtual machines to create.                                |    No    | `4`             |
| **maxSurge<sup>1</sup>**                  | int    | Specifies the maximum number of virtual machines that are created during an update.        |    No    | `4`             |
| **maxUnavailable<sup>1</sup>**            | int    | Specifies the maximum number of virtual machines that can be unavailable during an update. |    No    | `1`             |
| **additionalWorkerNodePools<sup>1</sup>** | array  | Defines a custom list of additional worker node pools.                                     |    No    | None            |


</details>

You can configure the following provisioning parameters for SAP Cloud Infrastructure:

<details>
<summary label="sap-converged-cloud-plan">
SAP Cloud Infrastructure
</summary>

| Parameter name                            | Type   | Description                                                                                | Required | Default value |
|-------------------------------------------|--------|--------------------------------------------------------------------------------------------|:--------:|---------------|
| **machineType**                           | string | Specifies the provider-specific virtual machine type.                                      |    No    | `g_c2_m8`     |
| **volumeSizeGb**                          | int    | Specifies the size of the root volume.                                                     |    No    | `30`          |
| **region**                                | string | Defines the cluster region.                                                                |   Yes    | None          |
| **zones**                                 | string | Defines the list of zones in which KIM creates a cluster.                                  |    No    | `["a"]`       |
| **autoScalerMin<sup>1</sup>**             | int    | Specifies the minimum number of virtual machines to create.                                |    No    | `3`           |
| **autoScalerMax<sup>1</sup>**             | int    | Specifies the maximum number of virtual machines to create.                                |    No    | `20`          |
| **maxSurge<sup>1</sup>**                  | int    | Specifies the maximum number of virtual machines that are created during an update.        |    No    | `4`           |
| **maxUnavailable<sup>1</sup>**            | int    | Specifies the maximum number of virtual machines that can be unavailable during an update. |    No    | `1`           |
| **additionalWorkerNodePools<sup>1</sup>** | array  | Defines a custom list of additional worker node pools.                                     |    No    | None          |


</details>

## Trial Plan

The trial plan allows you to install Kyma runtime on Azure, AWS, or Google Cloud. The plan assumptions are as follows:

* Kyma runtime is uninstalled after 14 days and the Kyma cluster is deprovisioned after this time.
* It's possible to provision only one Kyma runtime per global account.

### Provisioning Parameters

You can configure the following provisioning parameters for the Trial plan:

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

The **region** parameter is optional. If not specified, the region is calculated from the platform region specified in this path:

```shell
/oauth/{platform-region}/v2/service_instances/{instance_id}
```

The mapping between the platform region and the provider region (Azure, AWS or Google Cloud) is defined in the configuration file in the **APP_TRIAL_REGION_MAPPING_FILE_PATH** environment variable. If the platform region is not defined, the default value is `europe`.

</details>

## Preview Cluster Plan

The preview plan is designed for testing major changes in KEB's architecture.

### Provisioning Parameters

You can configure the following provisioning parameters for the `preview` plan:

<details>
<summary label="preview_cluster-plan">
Preview cluster plan
</summary>

| Parameter name                            | Type   | Description                                                                                | Required | Default value |
|-------------------------------------------|--------|--------------------------------------------------------------------------------------------|:--------:|---------------|
| **machineType**                           | string | Specifies the provider-specific virtual machine type.                                      |    No    | `m6i.large`   |
| **volumeSizeGb**                          | int    | Specifies the size of the root volume.                                                     |    No    | `50`          |
| **region**                                | string | Defines the cluster region.                                                                |   Yes    | None          |
| **zones**                                 | string | Defines the list of zones in which KIM creates a cluster.                                  |    No    | `["1"]`       |
| **autoScalerMin<sup>1</sup>**             | int    | Specifies the minimum number of virtual machines to create.                                |    No    | `3`           |
| **autoScalerMax<sup>1</sup>**             | int    | Specifies the maximum number of virtual machines to create, up to `40` allowed.            |    No    | `10`          |
| **maxSurge<sup>1</sup>**                  | int    | Specifies the maximum number of virtual machines that are created during an update.        |    No    | `4`           |
| **maxUnavailable<sup>1</sup>**            | int    | Specifies the maximum number of virtual machines that can be unavailable during an update. |    No    | `1`           |
| **additionalWorkerNodePools<sup>1</sup>** | array  | Defines a custom list of additional worker node pools.                                     |    No    | None          |

</details>

<br>
<p><sup>1</sup> This parameter is available for both provisioning and update operations.</p>

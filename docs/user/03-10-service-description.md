<!--{"metadata":{"publish":true}}-->

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
| `gdch`                   | `024e11fb-40df-4753-a992-24d136e2d15c` | Installs Kyma runtime in the GDCH cluster.                     |

There is also an experimental `preview` plan:

| Plan name | Plan ID                                | Description                                                      |
|-----------|----------------------------------------|------------------------------------------------------------------|
| `preview` | `5cb3d976-b85c-42ea-a636-79cadda109a9` | Installs Kyma runtime on AWS using Kyma Lifecycle Manager (KLM). |

> ### Caution:
> The experimental plan may fail to work or be removed.

## Provisioning Parameters

There are two types of configurable provisioning parameters: the ones that are compliant for all providers and provider-specific ones.

For internal KEB parameters not exposed through the OSB API, see [Internal Provisioning Parameters](../contributor/02-20-internal-parameters.md).

### Parameters Compliant for All Providers

You can configure the following provisioning parameters:

| Parameter name                                   | Type   | Description                                                                                 | Required | Default value                                                                                            |
|--------------------------------------------------|--------|---------------------------------------------------------------------------------------------|:--------:|----------------------------------------------------------------------------------------------------------|
| **name<sup>1</sup>**                             | string | Specifies the name of the cluster.                                                          |   Yes    | None                                                                                                     |
| **oidc.clientID<sup>1</sup>**                    | string | Provides an OIDC client ID for a Kyma runtime.                                              |    No    | None                                                                                                     |
| **oidc.groupsClaim<sup>1</sup>**                 | string | Provides an OIDC groups claim for a Kyma runtime.                                           |    No    | `groups`                                                                                                 |
| **oidc.issuerURL<sup>1</sup>**                   | string | Provides an OIDC issuer URL for a Kyma runtime.                                             |    No    | None                                                                                                     |
| **oidc.signingAlgs<sup>1</sup>**                 | string | Provides the OIDC signing algorithms for a Kyma runtime.                                    |    No    | `RS256`                                                                                                  |
| **oidc.usernameClaim<sup>1</sup>**               | string | Provides an OIDC username claim for a Kyma runtime.                                         |    No    | `email`                                                                                                  |
| **oidc.usernamePrefix<sup>1</sup>**              | string | Provides an OIDC username prefix for a Kyma runtime.                                        |    No    | None                                                                                                     |
| **oidc.encodedJwksArray<sup>1</sup>**            | string | Provides the JWKS array encoded in base64. To remove a previously set value, enter `-`.     |    No    | None                                                                                                     |
| **oidc.groupsPrefix<sup>1</sup>**                | string | Provides a prefix for group name claim mappings.                                            |    No    | None                                                                                                     |
| **oidc.requiredClaims<sup>1</sup>**              | array  | Provides a list of `key=value` pairs that describe required claims in the ID Token.         |    No    | None                                                                                                     |
| **administrators<sup>1</sup>**                   | string | Provides administrators for a Kyma runtime.                                                 |    No    | If no other value is provided, the email address of the provisioning user is used                        |
| **additionalWorkerNodePools<sup>1</sup>**        | array  | Defines a custom list of additional worker node pools.                                      |    No    | None                                                                                                     |
| **networking.nodes**                             | string | The CIDR range for nodes. Required when `networking` is specified.                          |    No    | `10.250.0.0/16`                                                                                          |
| **networking.pods**                              | string | The CIDR range for Pods.                                                                    |    No    | `10.96.0.0/13`                                                                                           |
| **networking.services**                          | string | The CIDR range for services.                                                                |    No    | `10.104.0.0/13`                                                                                          |
| **networking.dualStack**                         | bool   | Enables dual-stack networking. Available for AWS only.                                      |    No    | `false`                                                                                                  |
| **modules.channel**                              | string | Defines your preferred default release channel.                                             |    No    | Taken from the runtimeConfiguration setting, where the Kyma resource spec channel is specified per plan. |
| **modules.default**                              | bool   | Defines whether to use a default list of modules.                                           |    No    | None                                                                                                     |
| **modules.list**                                 | array  | Defines a custom list of modules.                                                           |    No    | None                                                                                                     |

### Provider-Specific Parameters

You can configure the following provisioning parameters for Microsoft Azure:

<details>
<summary label="azure-plan">
Microsoft Azure
</summary>

| Parameter name                         | Type   | Description                                                                     | Required | Default value     |
|----------------------------------------|--------|---------------------------------------------------------------------------------|:--------:|-------------------|
| **machineType<sup>1</sup>**            | string | Specifies the provider-specific virtual machine type.                           |    No    | Depends on configuration. See [Allowed Machine Types](https://github.com/kyma-project/kyma-environment-broker/blob/main/docs/contributor/03-70-machines-configuration.md#allowed-machine-types). |
| **additionalVolumeSizeGi<sup>1</sup>** | int    | Specifies extra disk space on top of the default volume size.                   |    No    | None              |
| **region**                             | string | Defines the cluster region.                                                     |   Yes    | None              |
| **autoScalerMin<sup>1</sup>**          | int    | Specifies the minimum number of virtual machines to create.                     |    No    | `3`               |
| **autoScalerMax<sup>1</sup>**          | int    | Specifies the maximum number of virtual machines to create, up to `40` allowed. |    No    | `20`              |
| **colocateControlPlane**               | bool   | Colocates both the control plane and worker nodes in the same region.           |    No    | `false`           |
| **accessControlList<sup>1</sup>**      | object | Specifies the IP ranges that can access the Kubernetes API.                     |    No    | None              |
| **auditLogAccess<sup>1</sup>**         | bool   | Enables direct read access to audit log data.                                   |    No    | `false`           |
<!-- TODO: confirm whether gvisor and ingressFiltering should be included in user-facing docs (internal users only) -->
| **gvisor.enabled<sup>1</sup>**         | bool   | Enables gVisor sandbox for workloads.                                           |    No    | `false`           |
| **ingressFiltering<sup>1</sup>**       | bool   | Controls ingress traffic filtering.                                             |    No    | `false`           |

</details>

<details>
<summary label="azure-lite-plan">
Azure Lite
</summary>

| Parameter name                 | Type   | Description                                                                     | Required | Default value     |
|--------------------------------|--------|---------------------------------------------------------------------------------|:--------:|-------------------|
| **machineType<sup>1</sup>**    | string | Specifies the provider-specific virtual machine type.                           |    No    | Depends on configuration. See [Allowed Machine Types](https://github.com/kyma-project/kyma-environment-broker/blob/main/docs/contributor/03-70-machines-configuration.md#allowed-machine-types). |
| **region**                     | string | Defines the cluster region.                                                     |   Yes    | None              |
| **autoScalerMin<sup>1</sup>**  | int    | Specifies the minimum number of virtual machines to create.                     |    No    | `3`               |
| **autoScalerMax<sup>1</sup>**  | int    | Specifies the maximum number of virtual machines to create, up to `40` allowed. |    No    | `20`              |
| **colocateControlPlane**       | bool   | Colocates both the control plane and worker nodes in the same region.           |    No    | `false`           |
| **auditLogAccess<sup>1</sup>** | bool   | Enables direct read access to audit log data.                                   |    No    | `false`           |
<!-- TODO: confirm whether gvisor should be included in user-facing docs (internal users only) -->
| **gvisor.enabled<sup>1</sup>** | bool   | Enables gVisor sandbox for workloads.                                           |    No    | `false`           |

</details>

You can configure the following provisioning parameters for Amazon Web Services (AWS):

<details>
<summary label="aws-plan">
AWS
</summary>

| Parameter name                         | Type   | Description                                                                     | Required | Default value |
|----------------------------------------|--------|---------------------------------------------------------------------------------|:--------:|---------------|
| **machineType<sup>1</sup>**            | string | Specifies the provider-specific virtual machine type.                           |    No    | Depends on configuration. See [Allowed Machine Types](https://github.com/kyma-project/kyma-environment-broker/blob/main/docs/contributor/03-70-machines-configuration.md#allowed-machine-types). |
| **additionalVolumeSizeGi<sup>1</sup>** | int    | Specifies extra disk space on top of the default volume size.                   |    No    | None          |
| **region**                             | string | Defines the cluster region.                                                     |   Yes    | None          |
| **autoScalerMin<sup>1</sup>**          | int    | Specifies the minimum number of virtual machines to create.                     |    No    | `3`           |
| **autoScalerMax<sup>1</sup>**          | int    | Specifies the maximum number of virtual machines to create, up to `40` allowed. |    No    | `20`          |
| **colocateControlPlane**               | bool   | Colocates both the control plane and worker nodes in the same region.           |    No    | `false`       |
| **accessControlList<sup>1</sup>**      | object | Specifies the IP ranges that can access the Kubernetes API.                     |    No    | None          |
| **auditLogAccess<sup>1</sup>**         | bool   | Enables direct read access to audit log data.                                   |    No    | `false`       |
<!-- TODO: confirm whether gvisor and ingressFiltering should be included in user-facing docs (internal users only) -->
| **gvisor.enabled<sup>1</sup>**         | bool   | Enables gVisor sandbox for workloads.                                           |    No    | `false`       |
| **ingressFiltering<sup>1</sup>**       | bool   | Controls ingress traffic filtering.                                             |    No    | `false`       |

</details>

You can configure the following provisioning parameters for Google Cloud:

<details>
<summary label="gcp-plan">
Google Cloud
</summary>

| Parameter name                         | Type   | Description                                                           | Required | Default value   |
|----------------------------------------|--------|-----------------------------------------------------------------------|:--------:|-----------------|
| **machineType<sup>1</sup>**            | string | Specifies the provider-specific virtual machine type.                 |    No    | Depends on configuration. See [Allowed Machine Types](https://github.com/kyma-project/kyma-environment-broker/blob/main/docs/contributor/03-70-machines-configuration.md#allowed-machine-types). |
| **additionalVolumeSizeGi<sup>1</sup>** | int    | Specifies extra disk space on top of the default volume size.         |    No    | None            |
| **region**                             | string | Defines the cluster region.                                           |   Yes    | None            |
| **autoScalerMin<sup>1</sup>**          | int    | Specifies the minimum number of virtual machines to create.           |    No    | `3`             |
| **autoScalerMax<sup>1</sup>**          | int    | Specifies the maximum number of virtual machines to create.           |    No    | `20`            |
| **colocateControlPlane**               | bool   | Colocates both the control plane and worker nodes in the same region. |    No    | `false`         |
| **auditLogAccess<sup>1</sup>**         | bool   | Enables direct read access to audit log data.                         |    No    | `false`         |
<!-- TODO: confirm whether gvisor and ingressFiltering should be included in user-facing docs (internal users only) -->
| **gvisor.enabled<sup>1</sup>**         | bool   | Enables gVisor sandbox for workloads.                                 |    No    | `false`         |
| **ingressFiltering<sup>1</sup>**       | bool   | Controls ingress traffic filtering.                                   |    No    | `false`         |

</details>

You can configure the following provisioning parameters for SAP Cloud Infrastructure:

<details>
<summary label="sap-converged-cloud-plan">
SAP Cloud Infrastructure
</summary>

| Parameter name                         | Type   | Description                                                           | Required | Default value |
|----------------------------------------|--------|-----------------------------------------------------------------------|:--------:|---------------|
| **machineType<sup>1</sup>**            | string | Specifies the provider-specific virtual machine type.                 |    No    | Depends on configuration. See [Allowed Machine Types](https://github.com/kyma-project/kyma-environment-broker/blob/main/docs/contributor/03-70-machines-configuration.md#allowed-machine-types). |
| **additionalVolumeSizeGi<sup>1</sup>** | int    | Specifies extra disk space on top of the default volume size.         |    No    | None          |
| **region**                             | string | Defines the cluster region.                                           |   Yes    | None          |
| **autoScalerMin<sup>1</sup>**          | int    | Specifies the minimum number of virtual machines to create.           |    No    | `3`           |
| **autoScalerMax<sup>1</sup>**          | int    | Specifies the maximum number of virtual machines to create.           |    No    | `20`          |
| **colocateControlPlane**               | bool   | Colocates both the control plane and worker nodes in the same region. |    No    | `false`       |
| **auditLogAccess<sup>1</sup>**         | bool   | Enables direct read access to audit log data.                         |    No    | `false`       |

</details>

You can configure the following provisioning parameters for Google Distributed Cloud Hosted (GDCH):

<details>
<summary label="gdch-plan">
Google Distributed Cloud Hosted
</summary>

> ### Note:
> The custom **networking** configuration parameter is not available for the `gdch` plan and is disabled by design.

| Parameter name                 | Type   | Description                                                                                | Required | Default value       |
|--------------------------------|--------|--------------------------------------------------------------------------------------------|:--------:|---------------------|
| **machineType<sup>1</sup>**    | string | Specifies the provider-specific virtual machine type.                                      |    No    | Depends on configuration. See [Allowed Machine Types](https://github.com/kyma-project/kyma-environment-broker/blob/main/docs/contributor/03-70-machines-configuration.md#allowed-machine-types). |
| **region**                     | string | Defines the cluster region.                                                                |   Yes    | None                |
| **autoScalerMin<sup>1</sup>**  | int    | Specifies the minimum number of virtual machines to create.                                |    No    | `3`                 |
| **autoScalerMax<sup>1</sup>**  | int    | Specifies the maximum number of virtual machines to create.                                |    No    | `10`                |

</details>

You can configure the following provisioning parameters for Alibaba Cloud:

<details>
<summary label="alicloud-plan">
Alibaba Cloud
</summary>

| Parameter name                | Type   | Description                                                           | Required | Default value   |
|-------------------------------|--------|-----------------------------------------------------------------------|:--------:|-----------------|
| **machineType<sup>1</sup>**   | string | Specifies the provider-specific virtual machine type.                 |    No    | Depends on configuration. See [Allowed Machine Types](https://github.com/kyma-project/kyma-environment-broker/blob/main/docs/contributor/03-70-machines-configuration.md#allowed-machine-types). |
| **region**                    | string | Defines the cluster region.                                           |   Yes    | None            |
| **autoScalerMin<sup>1</sup>** | int    | Specifies the minimum number of virtual machines to create.           |    No    | `3`             |
| **autoScalerMax<sup>1</sup>** | int    | Specifies the maximum number of virtual machines to create.           |    No    | `20`            |
| **colocateControlPlane**      | bool   | Colocates both the control plane and worker nodes in the same region. |    No    | `false`         |
<!-- TODO: confirm whether ingressFiltering should be included in user-facing docs (internal users only) -->
| **ingressFiltering<sup>1</sup>** | bool | Controls ingress traffic filtering.                                  |    No    | `false`         |

</details>

## Trial Plan

The trial plan allows you to install Kyma runtime on Azure, AWS, or Google Cloud. The plan assumptions are as follows:

* Kyma runtime is uninstalled after 14 days and the Kyma cluster is deprovisioned after this time.
* It's possible to provision only one Kyma runtime per global account.

### Provisioning Parameters

You can configure the following provisioning parameters for the trial plan:

<details>
<summary label="trial-plan">
Trial plan
</summary>

| Parameter name | Type   | Description                                                    | Required | Possible values       | Default value                       |
|----------------|--------|----------------------------------------------------------------|----------|-----------------------|-------------------------------------|
| **provider**   | string | Specifies the cloud provider used during provisioning.         | No       | `Azure`, `AWS`, `GCP` | Depends on the deployment configuration |
| **region**     | string | Defines the cluster region.                                    | No       | `europe`, `us`, `asia` | Calculated from the platform region |

The **region** parameter is optional. If not specified, the region is calculated from the platform region specified in this path:

```shell
/oauth/{platform-region}/v2/service_instances/{instance_id}
```

The mapping between the platform region and the provider region is defined in the configuration file in the **APP_TRIAL_REGION_MAPPING_FILE_PATH** environment variable.

</details>

## Preview Cluster Plan

The preview plan is designed for testing major changes in KEB's architecture.

### Provisioning Parameters

You can configure the following provisioning parameters for the `preview` plan:

<details>
<summary label="preview_cluster-plan">
Preview cluster plan
</summary>

| Parameter name                | Type   | Description                                                                     | Required | Default value |
|-------------------------------|--------|---------------------------------------------------------------------------------|:--------:|---------------|
| **machineType<sup>1</sup>**   | string | Specifies the provider-specific virtual machine type.                           |    No    | Depends on configuration. See [Allowed Machine Types](https://github.com/kyma-project/kyma-environment-broker/blob/main/docs/contributor/03-70-machines-configuration.md#allowed-machine-types). |
| **region**                    | string | Defines the cluster region.                                                     |   Yes    | None          |
| **autoScalerMin<sup>1</sup>** | int    | Specifies the minimum number of virtual machines to create.                     |    No    | `3`           |
| **autoScalerMax<sup>1</sup>** | int    | Specifies the maximum number of virtual machines to create, up to `40` allowed. |    No    | `10`          |

</details>

<br>
<p><sup>1</sup> This parameter is available for both provisioning and update operations.</p>

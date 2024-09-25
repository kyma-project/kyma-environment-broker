# Kyma Runtime End-to-End Provisioning Test

## Overview

Kyma runtime end-to-end provisioning test checks if [runtime provisioning](https://github.com/kyma-project/kyma-environment-broker/blob/main/docs/user/01-10-architecture.md) works as expected. The test is based on the implementation of the Kyma Environment Broker (KEB) and Runtime Provisioner. External dependencies relevant to this scenario are mocked.

The test is executed on a dev cluster. It is executed after every merge to the `kyma` repository that changes the `compass` chart.

## Prerequisites

- Gardener Secret per provider
- Service Manager Secret
- Kyma Environment Broker [configured](../../../docs/contributor/01-10-authorization.md) to use these Secrets to create Kyma runtime successfully

## Details

### End-to-End Provisioning Test
The provisioning end-to-end test contains a broker-client implementation that mocks the Registry. It is an external dependency that calls the broker in the regular scenario. The test is divided into two phases: provisioning and cleanup.

#### Provisioning

During the provisioning phase, the test performs the following steps:

1. Sends a call to KEB to provision a Kyma runtime. KEB creates the operation and sends a request to Runtime Provisioner. The test waits until the operation is successful. It takes about 30 minutes on Google Cloud and a few hours on Azure. You can configure the timeout using the environment variable.

2. Creates a ConfigMap with **instanceId** specified.

3. Fetches the DashboardURL from KEB. To do so, the runtime must be successfully provisioned and registered in the Director.

4. Updates the ConfigMap with **dashboardUrl** field.

5. Creates a Secret with a kubeconfig of the provisioned runtime.

6. Ensures that the DashboardURL redirects to the UUA login page, which means that Kyma runtime is accessible.

#### Cleanup

The cleanup logic is executed at the end of the end-to-end test or when the provisioning phase fails. During the cleanup, the test performs the following steps:

1. Gets **instanceId** from the ConfigMap.

2. Removes the test's Secret and ConfigMap.

3. Fetches the runtime kubeconfig from Runtime Provisioner and uses it to clean up resources that block the cluster from being deprovisioned.

4. Sends a request to deprovision the runtime to KEB. The request is passed to Runtime Provisioner, which deprovisions the runtime.

5. Waits until the deprovisioning is successful. It takes about 20 minutes to complete. You can configure the timeout using the environment variable.

Between the end-to-end test phases, you can run your own test directly on the provisioned runtime. To do so, use a kubeconfig stored in a Secret created in the provisioning phase.

### End-to-End Suspension Test

The end-to-end suspension test uses the **Trial** service plan ID to provision Kyma runtime. Then, the test suspends and unsuspends the Kyma runtime and ensures it is still accessible after the process. The suspension test works similarly to the provisioning test, but it has two additional steps in the provisioning phase:

#### Suspension

After successfully provisioning a Kyma runtime, the test sends an update call to KEB to suspend the runtime. Then, the test waits until the operation is successful.


#### Unsuspension

   After the runtime suspension succeeds, the test sends an update call to KEB to unsuspend it. Then, the test waits until the operation is successful. After that, the test ensures that the DashboardURL redirects to the UUA login page once again, which means the Kyma runtime is accessible.

After the Kyma runtime is successfully suspended and unsuspended, the test proceeds to the cleanup phase.

## Configuration

You can configure the test execution by using the following environment variables:

| Name | Description | Default value |
|-----|---------|:--------:|
| **APP_BROKER_URL** | Specifies the KEB URL. | None |
| **APP_PROVISION_TIMEOUT** | Specifies a timeout for the provisioning operation to succeed. | `3h` |
| **APP_DEPROVISION_TIMEOUT** | Specifies a timeout for the deprovisioning operation to succeed. | `1h` |
| **APP_BROKER_PROVISION_GCP** | Specifies if a runtime cluster is hosted on Google Cloud. If set to `false`, it is provisioned on Azure. | `true` |
| **APP_BROKER_AUTH_USERNAME** | Specifies the username for the basic authentication in KEB. | `broker` |
| **APP_BROKER_AUTH_PASSWORD** | Specifies the password for the basic authentication in KEB. | None |
| **APP_RUNTIME_PROVISIONER_URL** | Specifies the Runtime Provisioner URL. | None |
| **APP_RUNTIME_UUA_INSTANCE_NAME** | Specifies the name of the UUA instance which is provisioned in the Runtime. | `uua-issuer` |
| **APP_RUNTIME_UUA_INSTANCE_NAMESPACE** | Specifies the namespace of the UUA instance which is provisioned in the runtime. | `kyma-system` |
| **APP_TENANT_ID** | Specifies TenantID which is used in the test. | None |
| **APP_DUMMY_TEST** | Specifies if the test should succeed without any action. | `false` |
| **APP_CLEANUP_PHASE** | Specifies if the test runs the cleanup phase. | `false` |
| **APP_CONFIG_NAME** | Specifies the name of the ConfigMap and Secret created in the test. | `e2e-runtime-config` |
| **APP_DEPLOY_NAMESPACE** | Specifies the namespace of the ConfigMap and Secret created in the test. | `kcp-system` |
| **APP_BUSOLA_URL** | Specifies the URL to the expected Kyma dashboard used when asserting redirection to the UI Console.  | `kcp-system` |

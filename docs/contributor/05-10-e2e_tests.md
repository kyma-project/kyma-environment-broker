# End-to-end Tests of Kyma Environment Broker

## Overview

The following end-to-end (E2E) tests cover Kyma Environment Broker (KEB) and SAP BTP, Kyma runtime:

* `skr-tests` for testing the following operations on different cloud service providers: Kyma provisioning, BTP Manager Secret reconciliation, updating OIDC, updating machine type, and Kyma runtime deprovisioning
* `keb-endpoints-test` for checking if `kyma-environment-broker` endpoints require authorization
* `skr-aws-networking` for checking if provisioning a Kyma runtime with custom networking parameters works as expected
* `skr-trial-suspension-dev` for testing the following operations: Kyma provisioning, Kyma suspension, and Kyma runtime deprovisioning
* `skr-aws-binding` for testing the following operations: Kyma provisioning, fetching Kyma Binding, using Kyma Binding, deleting Kyma Binding, and Kyma runtime deprovisioning
* `provisioning-service-aws-stage` for checking if Cloud Management Service Provisioning API works as expected

## E2E SKR Tests

### Usage

You can use the SKR test in two modes - with or without provisioning.

#### With Provisioning

In this mode, the test executes the following steps:

1. Provisions a Kyma runtime cluster.
2. Runs the BTP Manager Secret reconciliation test.
3. Runs the OIDC test.
4. Runs the machine type update test.
5. Deprovisions the Kyma runtime instance and cleans up the resources.

#### Without Provisioning

In this mode the test additionally needs the following environment variables:

* **SKIP_PROVISIONING** set to `true`
* **INSTANCE_ID** - the UUID of the provisioned Kyma runtime instance

In this mode, the test executes the following steps:

1. Ensures the Kyma runtime exists.
2. Runs the OIDC test.
3. Cleans up the resources.

### Test Execution

1. Before you run the test, prepare the `.env` file based on this [`.env.template`](/testing/e2e/skr/skr-test/.env.template).
2. To set up the environment variables in your system, run:

    ```bash
    export $(xargs < .env)
    ```

3. Choose whether you want to run the test with or without provisioning.

    * To run the test with provisioning, call the following target:

        ```bash
        make skr
        ```

    * To run the SKR test without provisioning, use the following command:

        ```bash
        make skr SKIP_PROVISIONING=true
        ```

## KEB Endpoints Test

### Usage

The test executes the following steps:
1. Calls KEB endpoints without an authorization token.
2. Checks whether the call was rejected.

### Test Execution

1. Before you run the test, prepare the `.env` file based on this [`.env.template`](/testing/e2e/skr/keb-endpoints-test/.env.template).
2. To set up the environment variables in your system, run:

    ```bash
    export $(xargs < .env)
    ```

3. Run the test scenario.
   
    ```bash
    make keb-endpoints
    ```

## Networking Parameter Tests

### Usage

The test executes the following steps:

1. Calls KEB endpoints with invalid networking parameters.
2. Checks whether the call was rejected.
3. Provisions a cluster with custom networking parameters.
4. Deprovisions the cluster.

### Test Execution

1. Before you run the test, prepare the `.env` file based on this [`.env.template`](/testing/e2e/skr/skr-networking-test/.env.template).
2. To set up the environment variables in your system, run:

    ```bash
    export $(xargs < .env)
    ```

3. Run the test scenario:

    ```bash
    make skr-networking
    ```

## E2E SKR Suspension Test

### Usage

The test executes the following steps:

1. Provisions a Kyma runtime cluster.
2. Waits until Trial Cleanup CronJob triggers suspension.
3. Waits until suspension succeeds.
4. Deprovisions the Kyma runtime instance and cleans up the resources.

### Test Execution

1. Before you run the test, prepare the `.env` file based on this [`.env.template`](/testing/e2e/skr/skr-test/.env.template).
2. To set up the environment variables in your system, run:

    ```bash
    export $(xargs < .env)
    ```

3. Run the test scenario:

    ```bash
    make skr-trial-suspension
    ```

## Binding Tests

### Usage

The test executes the following steps:

1. Provisions a Kyma runtime cluster.
2. Creates a Kyma Binding and saves the returned kubeconfig.
3. Initializes a Kubernetes client with the returned kubeconfig.
4. Fetches the `sap-btp-manager` Secret using the Kyma Binding.
5. Fetches the created Kyma Binding.
6. Deletes the created Kyma Binding.
7. Tries to fetch the `sap-btp-manager` Secret using the deleted Kyma Binding.
8. Tries to create a Kyma Binding using invalid parameters.
9. Tests response status codes.
10. Tries to create more than 10 Kyma Bindings.
11. Deprovisions the Kyma runtime instance and cleans up the resources.

### Test Execution

1. Before you run the test, prepare the `.env` file based on this [`.env.template`](/testing/e2e/skr/skr-test/.env.template).
2. To set up the environment variables in your system, run:

    ```bash
    export $(xargs < .env)
    ```

3. Run the test scenario:

    ```bash
    make skr-binding
    ```

## Provisioning Service Tests

### Usage

The test executes the following steps:

0. Sends a call to Provisioning API to deprovision the remaining Kyma runtime and waits until the environment is deleted if the previous test run was not able to deprovision Kyma runtime.
1. Sends a call to Provisioning API to provision a Kyma runtime. The test waits until the environment is created.
2. Creates a Kyma Binding.
3. Fetches the `sap-btp-manager` Secret using the kubeconfig from the created Kyma Binding.
4. Fetches the created Kyma Biding.
5. Deletes the created Kyma Binding.
6. Tries to fetch the `sap-btp-manager` Secret using the invalidated kubeconfig.
7. Tries to fetch the deleted Kyma Binding.
8. Sends a call to Provisioning API to deprovision the Kyma runtime. The test waits until the environment is deleted.

### Test Execution

1. Before you run the test, prepare the `.env` file based on this [`.env.template`](/testing/e2e/skr/provisioning-service-test/.env.template).
2. To set up the environment variables in your system, run:

    ```bash
    export $(xargs < .env)
    ```

3. Run the test scenario:

    ```bash
    make provisioning-service
    ```

## CI Pipelines

The tests are run daily.

* `keb-endpoints-test` - KEB endpoints test
* `skr-aws-integration-dev` - SKR test
* `skr-aws-binding` - Kyma Bindings test
* `skr-aws-networking` - networking parameters test
* `skr-azure-integration-dev` - SKR test
* `skr-azure-lite-integration-dev` - SKR test
* `skr-free-aws-integration-dev` - SKR test
* `skr-preview-dev` - SKR test
* `skr-sap-converged-cloud-integration-dev` - SKR test
* `skr-trial-integration-dev` - SKR test
* `skr-trial-suspension-dev` - SKR suspension test
* `provisioning-service-aws-stage` - Provisioning API test

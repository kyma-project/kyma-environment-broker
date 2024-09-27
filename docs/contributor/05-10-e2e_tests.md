# End-to-end Tests of Kyma Environment Broker

## Overview

The end-to-end (E2E) tests cover Kyma Environment Broker (KEB) and SAP BTP, Kyma runtime.
There are three tests:
- `skr-tests` for testing the following operations on different cloud service providers: Kyma provisioning, BTP Manager Secret reconciliation, updating OIDC, updating machine type, and Kyma runtime deprovisioning
- `skr-aws-upgrade-integration` for checking Kyma runtime provisioning, upgrading, and deprovisioning
- `keb-endpoints-test` for checking if `kyma-environment-broker` endpoints require authorization

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
- **SKIP_PROVISIONING** set to `true`
- **INSTANCE_ID** - the UUID of the provisioned Kyma runtime instance

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
    - To run the test with provisioning, call the following target:

        ```bash
        make skr
        ```
    - To run the SKR test without provisioning, use the following command:

        ```bash
        make skr SKIP_PROVISIONING=true
        ```

## E2E SKR AWS Upgrade Integration Test

### Usage

The test executes the following steps:
1. Provisions a Kyma runtime cluster.
2. Runs a Kyma runtime upgrade.
3. Deprovisions the Kyma runtime instance and cleans up the resources.

### Test Execution 

1. Before you run the test, prepare the `.env` file based on this [`.env.template`](/testing/e2e/skr/skr-aws-upgrade-integration/.env.template).
2. To set up the environment variables in your system, run:

    ```bash
    export $(xargs < .env)
    ```

3. Run the test scenario:
   
    ```bash
    make skr-aws-upgrade-integration
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
    make skr-networking-test
    ```

## Binding Tests

### Usage

The test executes the following steps:
1. Provisions a Kyma runtime cluster.
2. Creates a binding using Kubernetes TokenRequest and saves the returned kubeconfig.
3. Initializes a Kubernetes client with the returned kubeconfig.
4. Tries to fetch a Secret using the binding from Kubernetes TokenRequest.
5. Creates a binding using Gardener and saves the returned kubeconfig.
6. Initializes a Kubernetes client with the returned kubeconfig.
7. Tries to fetch a Secret using the binding from Gardener.
8. Deprovisions the Kyma runtime instance and cleans up the resources.

### Test Execution

1. Before you run the test, prepare the `.env` file based on this [`.env.template`](/testing/e2e/skr/skr-test/.env.template).
2. To set up the environment variables in your system, run:

    ```bash
    export $(xargs < .env)
    ```

3. Run the test scenario:
    ```bash
    make skr-binding-test
    ```

## CI Pipelines

The tests are run once per day at 01:05 by the given ProwJobs:
- `skr-azure-integration-dev` - SKR test
- `skr-azure-lite-integration-dev` - SKR test
- `skr-trial-integration-dev` - SKR test
- `skr-preview-dev` - SKR test
- `skr-free-aws-integration-dev` - SKR test
- `skr-aws-integration-dev` - SKR test
- `skr-aws-upgrade-integration-dev` - SKR AWS upgrade integration test
- `keb-endpoints-test` - KEB endpoints test
- `skr-networking-test` - networking parameters test

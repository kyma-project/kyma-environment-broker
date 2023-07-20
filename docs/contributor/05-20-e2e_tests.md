# E2E tests of Kyma Environment Broker

## Overview

The end-to-end (E2E) tests cover Kyma Environment Broker (KEB) and SAP BTP, Kyma runtime (SKR).
There are three tests:
- `skr-tests` for testing the following operations on different cloud service providers: Kyma provisioning, BTP Manager secret reconciliation, updating OIDC, updating machine type, and Kyma deprovisioning
- `skr-aws-upgrade-integration` for checking Kyma provisioning, Kyma upgrading, and Kyma deprovisioning
- `keb-endpoints-test` for checking if kyma-environment-broker endpoints require authorization

## E2E SKR tests

### Usage

You can use the SKR test in two modes - with and without provisioning.

#### With provisioning

In this mode, the test executes the following steps:
1. Provisions an SKR cluster.
2. Runs the BTP Manager secret reconciliation test.
3. Runs the OIDC test.
4. Runs the machine type update test.
5. Deprovisions the SKR instance and cleans up the resources.

#### Without provisioning

In this mode the test additionally needs the following environment variables:
- **SKIP_PROVISIONING** set to `true`
- **INSTANCE_ID** - the UUID of the provisioned SKR instance

In this mode, the test executes the following steps:
1. Ensures the SKR exists.
2. Runs the OIDC test.
3. Cleans up the resources.
 
### Test execution

1. Before you run the test, prepare the `.env` file based on the following [`.env.template`](/testing/e2e/skr/skr-test/.env.template):
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

## E2E SKR AWS upgrade integration test

### Usage

The test executes the following steps:
1. Provisions an SKR cluster.
2. Runs Kyma upgrade.
3. Deprovisions the SKR instance and cleans up the resources.

### Test execution 

1. Before you run the test, prepare the `.env` file based on the following [`.env.template`](/testing/e2e/skr/skr-aws-upgrade-integration/.env.template):
2. To set up the environment variables in your system, run:

```bash
export $(xargs < .env)
```

3. Run the test scenario:
```bash
make skr-aws-upgrade-integration
```

## KEB endpoints test

### Usage

The test executes the following steps:
1. Calls KEB endpoints without an authorization token.
2. Checks whether the call was rejected.

### Test execution 

1. Before you run the test, prepare the `.env` file based on the following [`.env.template`](/testing/e2e/skr/keb-endpoints-test/.env.template):
2. To set up the environment variables in your system, run:

```bash
export $(xargs < .env)
```

3. Run the test scenario.
```bash
make keb-endpoints
```

### CI pipelines

The tests are run once per day at 01:05 by the given Prow jobs:
- `skr-azure-integration-dev` - SKR test
- `skr-azure-lite-integration-dev` - SKR test
- `skr-trial-integration-dev` - SKR test
- `skr-preview-dev` - SKR test
- `skr-free-aws-integration-dev` - SKR test
- `skr-aws-integration-dev` - SKR test
- `skr-aws-upgrade-integration-dev` - SKR AWS upgrade integration test
- `keb-endpoints-test` - KEB endpoints test

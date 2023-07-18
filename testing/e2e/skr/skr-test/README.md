# SKR test

This test covers SAP BTP, Kyma runtime (SKR).

## File structure
- `provision` folder contains the scripts for provisioning and de-provisioning the SKR cluster using KEB client
- `oidc` folder contains the OIDC-related tests

## Usage modes

You can use the SKR test in two modes - with and without provisioning.

### With provisioning

In this mode, the test executes the following steps:
1. Provisions an SKR cluster.
2. Runs the OIDC test.
3. Deprovisions the SKR instance and clean up the resources.

### Without provisioning

In this mode the test additionally needs the following environment variables:
- **SKIP_PROVISIONING** set to `true`
- **INSTANCE_ID** the UUID of the provisioned SKR instance

In this mode, the test executes the following steps:
1. Ensures the SKR exists.
2. Runs the OIDC test.
3. Cleans up the resources.
 
>**NOTE:** The SKR test additionally contains a stand-alone script that you can use to register the resources.

## Test execution

1. Before you run the test, prepare the `.env` file based on the following [.env.template](.env.template):
```
INSTANCE_ID
SKIP_PROVISIONING
KEB_HOST=
KEB_CLIENT_ID=
KEB_CLIENT_SECRET=
KEB_GLOBALACCOUNT_ID=
KEB_SUBACCOUNT_ID=
KEB_USER_ID=
KEB_PLAN_ID=
KEB_TOKEN_URL=
GARDENER_KUBECONFIG=

KCP_KEB_API_URL=
KCP_GARDENER_NAMESPACE=
KCP_OIDC_ISSUER_URL=
KCP_OIDC_CLIENT_ID=
KCP_OIDC_CLIENT_SECRET=
KCP_TECH_USER_LOGIN=
KCP_TECH_USER_PASSWORD=
KCP_MOTHERSHIP_API_URL=
KCP_KUBECONFIG_API_URL=

BTP_OPERATOR_CLIENTID=
BTP_OPERATOR_CLIENTSECRET=
BTP_OPERATOR_URL=
BTP_OPERATOR_TOKENURL=

AL_SERVICE_KEY= #must be a Cloud Foundry service key with the info about UAA (User Account and Authentication). To learn more about Managing Service Keys in Cloud Foundry, go to: https://docs.cloudfoundry.org/devguide/services/service-keys.html.
```

2. To set up the environment variables in your system, run:

```bash
export $(xargs < .env)
```

3. Choose whether you want to run the test with or without provisioning.
   - To run the test with provisioning, call the following target:

    ```bash
    npm run skr-test
    #or
    make skr
    ```
    - To run the SKR test without provisioning, use the following command:

    ```bash
    make skr SKIP_PROVISIONING=true
    #or
    npm run skr-test # when all env vars are exported
    ```


# Provision SAP BTP, Kyma Runtime Using SAP Cloud Management Service's Provisioning API

SAP Cloud Management Service (cis) provides Provisioning Service API to create and manage available environments. This document describes how to use the Provisioning Service API to create and access SAP BTP, Kyma runtime on AWS.

## Prerequisites
Your subaccount must have [entitlements](https://help.sap.com/docs/btp/sap-business-technology-platform/managing-entitlements-and-quotas-using-cockpit) for [SAP BTP, Kyma runtime](https://discovery-center.cloud.sap/index.html#/serviceCatalog/kyma-runtime) and [SAP Cloud Management Service for SAP BTP
](https://discovery-center.cloud.sap/index.html#/serviceCatalog/cloud-management-service).

### CLI tools
- [jq](https://jqlang.github.io/jq/)
- [curl](https://curl.se/)
- [btp CLI](https://help.sap.com/docs/btp/sap-business-technology-platform/download-and-start-using-btp-cli-client?locale=en-US) (optional)

## Steps

1. Provision SAP Cloud Management Service instance with `local` plan and create a binding to get the credentials for Provisioning Service API. The procedure is described in [Help Portal](https://help.sap.com/docs/btp/sap-business-technology-platform/getting-access-token-for-sap-cloud-management-service-apis) or you can use btp CLI as shown below:

   1. Set `CIS_INSTANCE_NAME` environment variable with the name of the SAP Cloud Management Service instance:
      ```bash
      export CIS_INSTANCE_NAME={CIS_INSTANCE_NAME}
      ```
   2. Provision SAP Cloud Management Service instance with Client Credentials grant type passed as parameters: 
      ```bash
      btp create services/instance --offering-name cis --plan-name local --name ${CIS_INSTANCE_NAME} --parameters {\"grantType\":\"clientCredentials\"}
      ```
   3. Create a binding for the instance:
      ```bash
      btp create services/binding --name ${CIS_INSTANCE_NAME}-binding --instance-name ${CIS_INSTANCE_NAME}
      ```

2. Set `CLIENT_ID`, `CLIENT_SECRET`, `UAA_URL`, `PROVISIONING_SERVICE_URL` environment variables using the credentials from the binding stored in `clientid`, `clientsecret`, `url`, `provisioning_service_url` fields. You can use btp CLI to get the credentials as shown below:
   ```bash
   export CLIENT_ID=$(btp --format json get services/binding --name cis-local-binding | jq -r '.credentials.uaa.clientid')
   export CLIENT_SECRET=$(btp --format json get services/binding --name cis-local-binding | jq -r '.credentials.uaa.clientsecret')
   export UAA_URL=$(btp --format json get services/binding --name cis-local-binding | jq -r '.credentials.uaa.url')
   export PROVISIONING_SERVICE_URL=$(btp --format json get services/binding --name cis-local-binding | jq -r '.credentials.endpoints.provisioning_service_url')
   ```

3. Get the access token for Provisioning Service API using the client credentials:
   ```bash
   TOKEN=$(curl -s -X POST "${UAA_URL}/oauth/token" -H "Content-Type: application/x-www-form-urlencoded" -u "${CLIENT_ID}:${CLIENT_SECRET}" --data-urlencode "grant_type=client_credentials" | jq -r '.access_token')
   ```

4. Check if Kyma runtime is available for provisioning:
   ```bash
   curl -s "$PROVISIONING_SERVICE_URL/provisioning/v1/availableEnvironments" -H "accept: application/json" -H "Authorization: bearer $TOKEN" | jq
   ```

   > [!NOTE]
   > **environmentType**, **planName**, and **serviceName** are required for the provisioning request.

5. Set `NAME`, `REGION`, `PLAN`, `USER_ID` environment variables:
   ```bash
   export NAME={RUNTIME_NAME}
   export REGION={CLUSTER_REGION}
   export PLAN={KYMA_RUNTIME_PLAN_NAME}
   export PLAN_ID={KYMA_RUNTIME_PLAN_ID}
   export USER_ID={USER_ID}
   ```

6. Provision the Kyma runtime and save the instance ID in `INSTANCE_ID` environment variable:
   ```bash
   INSTANCE_ID=$(curl -s -X POST "$PROVISIONING_SERVICE_URL/provisioning/v1/environments" -H "accept: application/json" -H "Authorization: bearer $TOKEN" -H "Content-Type: application/json" -d "{\"environmentType\":\"kyma\",\"parameters\":{\"name\":\"$NAME\",\"region\":\"$REGION\"},\"planName\":\"$PLAN\",\"serviceName\":\"kymaruntime\",\"user\":\"$USER_ID\"}" | jq -r '.id')
   ```

7. After the provisioning is completed, create a binding to get the kubeconfig:
   ```bash
   curl -s -X PUT "$PROVISIONING_SERVICE_URL/provisioning/v1/environments/$INSTANCE_ID/bindings" -H "accept: application/json" -H "Authorization: bearer $TOKEN" -H "Content-Type: application/json" -d "{\"serviceInstanceId\":\"$INSTANCE_ID\",\"planId\":\"$PLAN_ID\"}" | jq -r '.credentials.kubeconfig' > kubeconfig.yaml
   ```

8. Set `KUBECONFIG` environment variable to the path of the kubeconfig file to access the cluster through **kubectl**:
   ```bash
   export KUBECONFIG=kubeconfig.yaml
   ```

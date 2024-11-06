# Manage SAP BTP, Kyma Runtime Using the Provisioning API of the SAP Cloud Management Service

The SAP Cloud Management service (technical name: `cis`) provides the Provisioning Service API to create and manage available environments. Use the Provisioning Service API to manage and access SAP BTP, Kyma runtime on AWS.

## Prerequisites

* Your subaccount must have entitlements for SAP BTP, Kyma runtime and SAP Cloud Management Service for SAP BTP. See [Managing Entitlements and Quotas Using the Cockpit](https://help.sap.com/docs/btp/sap-business-technology-platform/managing-entitlements-and-quotas-using-cockpit?&version=Cloud).

* CLI tools

  * [kubectl](https://kubernetes.io/docs/reference/kubectl/)
  * [jq](https://jqlang.github.io/jq/)
  * [curl](https://curl.se/)
  * [btp CLI](https://help.sap.com/docs/btp/sap-business-technology-platform/download-and-start-using-btp-cli-client?locale=en-US) (optional)

## Steps

1. Use the btp CLI to provision the SAP Cloud Management service instance with the `local` plan and create a binding to get the credentials for the Provisioning Service API:

   1. Set the **CIS_INSTANCE_NAME** environment variable with the name of the SAP Cloud Management service instance:
   
      ```bash
      export CIS_INSTANCE_NAME={CIS_INSTANCE_NAME}
      ```
   
   2. Provision the SAP Cloud Management service instance with the Client Credentials grant type passed as parameters:
   
      ```bash
      btp create services/instance --offering-name cis --plan-name local --name ${CIS_INSTANCE_NAME} --parameters {\"grantType\":\"clientCredentials\"}
      ```
      
   3. Create a binding for the instance:
   
      ```bash
      btp create services/binding --name ${CIS_INSTANCE_NAME}-binding --instance-name ${CIS_INSTANCE_NAME}
      ```

   > [!NOTE]
   > For alternative methods, see [Getting an Access Token for SAP Cloud Management Service APIs](https://help.sap.com/docs/btp/sap-business-technology-platform/getting-access-token-for-sap-cloud-management-service-apis?&version=Cloud).

2. Set the **CLIENT_ID**, **CLIENT_SECRET**, **UAA_URL**, and **PROVISIONING_SERVICE_URL** environment variables using the credentials from the binding stored in the **clientid**, **clientsecret**, **url**, and **provisioning_service_url** fields. Use the btp CLI to get the credentials:

   ```bash
   export CLIENT_ID=$(btp --format json get services/binding --name ${CIS_INSTANCE_NAME}-binding | jq -r '.credentials.uaa.clientid')
   export CLIENT_SECRET=$(btp --format json get services/binding --name ${CIS_INSTANCE_NAME}-binding | jq -r '.credentials.uaa.clientsecret')
   export UAA_URL=$(btp --format json get services/binding --name ${CIS_INSTANCE_NAME}-binding | jq -r '.credentials.uaa.url')
   export PROVISIONING_SERVICE_URL=$(btp --format json get services/binding --name ${CIS_INSTANCE_NAME}-binding | jq -r '.credentials.endpoints.provisioning_service_url')
   ```

3. Get the access token for the Provisioning Service API using the client credentials:

   ```bash
   TOKEN=$(curl -s -X POST "${UAA_URL}/oauth/token" -H "Content-Type: application/x-www-form-urlencoded" -u "${CLIENT_ID}:${CLIENT_SECRET}" --data-urlencode "grant_type=client_credentials" | jq -r '.access_token')
   ```

4. Check if Kyma runtime is available for provisioning:

   ```bash
   curl -s "$PROVISIONING_SERVICE_URL/provisioning/v1/availableEnvironments" -H "accept: application/json" -H "Authorization: bearer $TOKEN" | jq
   ```

5. Set the **ENVIRONMENT_TYPE** and **SERVICE_NAME** environment variables to const values `kyma` and `kymaruntime`, and provide values for the **NAME**, **REGION**, **PLAN**, and **USER_ID** environment variables:

   ```bash
   export ENVIRONMENT_TYPE="kyma"
   export SERVICE_NAME="kymaruntime"
   export NAME={RUNTIME_NAME}
   export REGION={CLUSTER_REGION}
   export PLAN={KYMA_RUNTIME_PLAN_NAME}
   export USER_ID={USER_ID}
   ```

6. Provision the Kyma runtime and save the instance ID in **INSTANCE_ID** environment variable:

   ```bash
   INSTANCE_ID=$(curl -s -X POST "$PROVISIONING_SERVICE_URL/provisioning/v1/environments" -H "accept: application/json" -H "Authorization: bearer $TOKEN" -H "Content-Type: application/json" -d "{\"environmentType\":\"$ENVIRONMENT_TYPE\",\"parameters\":{\"name\":\"$NAME\",\"region\":\"$REGION\"},\"planName\":\"$PLAN\",\"serviceName\":\"$SERVICE_NAME\",\"user\":\"$USER_ID\"}" | jq -r '.id')
   ```

7. Optionally, set the **EXPIRATION_SECONDS** environment variable to the number of seconds (from 600 to 7200) after which a binding expires:

   ```bash
   export EXPIRATION_SECONDS={EXPIRATION_SECONDS}
   ```

8. After the provisioning is completed, create the binding to get the kubeconfig:

   ```bash
   [ -z "$EXPIRATION_SECONDS" ] && \
   curl -s -X PUT "$PROVISIONING_SERVICE_URL/provisioning/v1/environments/$INSTANCE_ID/bindings" -H "accept: application/json" -H "Authorization: bearer $TOKEN" | jq -r '.credentials.kubeconfig' > kubeconfig.yaml || \
   curl -s -X PUT "$PROVISIONING_SERVICE_URL/provisioning/v1/environments/$INSTANCE_ID/bindings" -H "accept: application/json" -H "Authorization: bearer $TOKEN" -H "Content-Type: application/json" -d "{\"parameters\":{\"expiration_seconds\":$EXPIRATION_SECONDS}}" | jq -r '.credentials.kubeconfig' > kubeconfig.yaml
   ```

9. To access the cluster through kubectl, set the **KUBECONFIG** environment variable to the path of the kubeconfig file:

   ```bash
   export KUBECONFIG=kubeconfig.yaml
   ```

10. Verify the connection to the cluster by running a kubectl command to get Pods:

    ```bash
    kubectl get pods
    ```

    kubectl should return the list of Pods in the `default` namespace running in the cluster, ehich means that the cluster is accessible.

> [!NOTE]
> The following steps are optional and show how to revoke the credentials by deleting the binding.

11. Get the ID of the binding from the instance's bindings list and save it in the **BINDING_ID** environment variable:

    ```bash
    curl -s "$PROVISIONING_SERVICE_URL/provisioning/v1/environments/$INSTANCE_ID/bindings" -H "accept: application/json" -H "Authorization: bearer $TOKEN"
    ```
   
    ```bash
    export BINDING_ID={BINDING_ID}
    ```

12. Delete the binding to revoke the credentials:

    ```bash
    curl -s -X DELETE "$PROVISIONING_SERVICE_URL/provisioning/v1/environments/$INSTANCE_ID/bindings/$BINDING_ID" -H "accept: application/json" -H "Authorization: bearer $TOKEN"
    ```

    Try to access the cluster using kubectl. The connection should be refused, which means that the binding was successfully deleted and credentials revoked:

    ```bash
    kubectl get pods
    ```

## Next Steps

Delete the instance bindings for the instance as shown in steps 11 and 12. To deprovision the Kyma runtime, run:

  ```bash
  curl -s -X DELETE "$PROVISIONING_SERVICE_URL/provisioning/v1/environments/$INSTANCE_ID" -H "accept: application/json" -H "Authorization: bearer $TOKEN"
  ```

> [!NOTE]
> The runtime can be deleted independently of the bindings. Existing bindings do not block the runtime deprovisioning.

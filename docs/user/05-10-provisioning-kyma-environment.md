# Provision SAP BTP, Kyma Runtime Using Kyma Environment Broker

Provision SAP BTP, Kyma runtime on Azure using Kyma Environment Broker (KEB).

## Steps

1. Export these values as environment variables:

   ```bash
   export BROKER_URL={KYMA_ENVIRONMENT_BROKER_URL}
   export INSTANCE_ID={INSTANCE_ID}
   export GLOBAL_ACCOUNT_ID={GLOBAL_ACCOUNT_ID}
   export NAME={RUNTIME_NAME}
   export USER_ID={USER_ID}
   export REGION={CLUSTER_REGION}
   ```

   > ### Note:
   > **INSTANCE_ID** and **NAME** must be unique. It is recommended to use UUID as an **INSTANCE_ID**.

2. Get the [access token](../contributor/01-10-authorization.md#get-the-access-token). Export this variable based on the token you got from the OAuth client:

   ```bash
   export AUTHORIZATION_HEADER="Authorization: Bearer $ACCESS_TOKEN"
   ```  

     Alternatively, you can perform `kubectl port-forward` on the chosen Pod to expose it on your local machine. Expose it on port `8080`:  

   ```bash
     kubectl port-forward -n kcp-system deployments/kcp-kyma-environment-broker 8080
   ```

3. Make a call to KEB to create a Kyma runtime on Azure. Find the list of possible request parameters in the [Service Description](03-10-service-description.md) document.

   ```bash
   curl --request PUT "https://$BROKER_URL/oauth/v2/service_instances/$INSTANCE_ID?accepts_incomplete=true" \
   --header 'X-Broker-API-Version: 2.14' \
   --header 'Content-Type: application/json' \
   --header "$AUTHORIZATION_HEADER" \
   --data-raw "{
       \"service_id\": \"47c9dcbf-ff30-448e-ab36-d3bad66ba281\",
       \"plan_id\": \"4deee563-e5ec-4731-b9b1-53b42d855f0c\",
       \"context\": {
           \"globalaccount_id\": \"$GLOBAL_ACCOUNT_ID\"
           \"user_id\": \"$USER_ID\"
       },
       \"parameters\": {
           \"name\": \"$NAME\",
           \"region\": \"$REGION\"
       }
   }"
   ```

   A successful call returns the operation ID:

    ```json
   {
       "operation":"8a7bfd9b-f2f5-43d1-bb67-177d2434053c"
   }
   ```  

4. Check the operation status as described in the [Check Operation Status](05-30-operation-status.md) document.

## SAP BTP Service Operator

If you need the SAP BTP service operator component installed, obtain the [SAP BTP service operator access credentials](https://github.com/SAP/sap-btp-service-operator/blob/v0.2.5/README.md#setup) and provide them in the provisioning request. See the following example:

 ```bash
   curl --request PUT "https://$BROKER_URL/oauth/v2/service_instances/$INSTANCE_ID?accepts_incomplete=true" \
   --header 'X-Broker-API-Version: 2.14' \
   --header 'Content-Type: application/json' \
   --header "$AUTHORIZATION_HEADER" \
   --data-raw "{
       \"service_id\": \"47c9dcbf-ff30-448e-ab36-d3bad66ba281\",
       \"plan_id\": \"4deee563-e5ec-4731-b9b1-53b42d855f0c\",
       \"context\": {
           \"globalaccount_id\": \"$GLOBAL_ACCOUNT_ID\",
           \"user_id\": \"$USER_ID\",
           \"sm_operator_credentials\": {
             \"clientid\": \"$clientid\",
             \"clientsecret\": \"$clientsecret\",
             \"sm_url\": \"$sm_url\",
             \"url\": \"$url\",
             \"xsappname\": \"$xsappname\"
		   },
       },
       \"parameters\": {
           \"name\": \"$NAME\",
           \"region\": \"$REGION\"
       }
   }"
   ```

```json
"sm_operator_credentials": {
  "clientid": "testClientID",
  "clientsecret": "testClientSecret",
  "sm_url": "https://service-manager.kyma.com",
  "url": "https://test.auth.com",
  "xsappname": "testXsappname"
}
```

# Cluster Name

Kyma Environment Broker (KEB) enables you to set a cluster name during SAP BTP, Kyma runtime provisioning and update operations. 
To do so, specify the **name** parameter in the provisioning or update request.
During provisioning, the **name** parameter is mandatory and must consist of 1–64 characters, using only letters, digits, or hyphens.
During update operations, the **name** parameter is optional. However, if provided, it must follow the same character and length requirements.
The cluster name is used as the context name when generating kubeconfig files for [Kyma Bindings](05-60-kyma-bindings.md) and the [Kubeconfig Endpoint](03-15-kubeconfig-endpoint.md).

See the example of the provisioning request:

```bash
   export VERSION=1.15.0
   curl --request PUT "https://$BROKER_URL/oauth/v2/service_instances/$INSTANCE_ID?accepts_incomplete=true" \
   --header 'X-Broker-API-Version: 2.14' \
   --header 'Content-Type: application/json' \
   --header "$AUTHORIZATION_HEADER" \
   --data-raw "{
       \"service_id\": \"47c9dcbf-ff30-448e-ab36-d3bad66ba281\",
       \"plan_id\": \"4deee563-e5ec-4731-b9b1-53b42d855f0c\",
       \"context\": {
           \"globalaccount_id\": \"$GLOBAL_ACCOUNT_ID\",
           \"subaccount_id\": \"$SUBACCOUNT_ID\",
           \"user_id\": \"$USER_ID\",
       },
       \"parameters\": {
           \"name\": \"$NAME\",
           \"region\": \"$REGION\"
       }
   }"
```

See the example of the update request:

```bash
   export VERSION=1.15.0
   curl --request PATCH "https://$BROKER_URL/oauth/v2/service_instances/$INSTANCE_ID?accepts_incomplete=true" \
   --header 'X-Broker-API-Version: 2.14' \
   --header 'Content-Type: application/json' \
   --header "$AUTHORIZATION_HEADER" \
   --data-raw "{
       \"service_id\": \"47c9dcbf-ff30-448e-ab36-d3bad66ba281\",
       \"plan_id\": \"4deee563-e5ec-4731-b9b1-53b42d855f0c\",
       \"context\": {
           \"globalaccount_id\": \"$GLOBAL_ACCOUNT_ID\",
           \"subaccount_id\": \"$SUBACCOUNT_ID\",
       },
       \"parameters\": {
           \"name\": \"$NAME\"
       }
   }"
```

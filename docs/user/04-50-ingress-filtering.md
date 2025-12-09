# Enable Ingress Filtering

> ### Note:
> The ingress filtering feature is only available to SAP internal customers as it is integrated with SAP's geo-blocking solution.
> The ingress filtering feature is available for the `aws`, `gcp`, and `azure` plans.

Kyma Environment Broker (KEB) allows you to enable or disable ingress filtering during SAP BTP, Kyma runtime provisioning and update operations.
By default, ingress filtering is disabled.
To enable it, set the additional **ingressFiltering** parameter to `true` in the provisioning or update request.

See the example:

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
           \"region\": \"$REGION\",
           \"ingressFiltering\": true
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
           \"ingressFiltering\": true
       }
   }"
```

> ### Note:
> Attempt to enable or disable ingress filtering for an unsupported plan or an external customer results in an error.

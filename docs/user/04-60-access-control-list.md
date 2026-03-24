# Access Control List

The Kyma Kubernetes API access can be restricted using Access Control List. You can specify IP ranges which are allowed to access the Kubernetes API. IP which does not belong to one of the range, are not allowed to access the API. Keep in mind that beside the IP address is in a specified range, the user must be authorized to access the API.
To define Access Control List you must provide `accessControlList` parameter in the provisioning request. For example:


```bash
   curl --request PUT "https://$BROKER_URL/oauth/v2/service_instances/$INSTANCE_ID?accepts_incomplete=true" \
   --header 'X-Broker-API-Version: 2.14' \
   --header 'Content-Type: application/json' \
   --header "$AUTHORIZATION_HEADER" \
   --header 'Content-Type: application/json' \
   --data-raw "{
       \"service_id\": \"47c9dcbf-ff30-448e-ab36-d3bad66ba281\",
       \"plan_id\": \"4deee563-e5ec-4731-b9b1-53b42d855f0c\",
       \"context\": {
           \"globalaccount_id\": \"$GLOBAL_ACCOUNT_ID\"
       },
       \"parameters\": {
           \"name\": \"$NAME\",
           \"region\": \"$REGION\",
           \"accessControlList\": {
               \"allowedCIDRs\": [\"1.2.3.0/24\", \"2.3.4.0/24\"]
           }
       }
   }"
```

The set of IP ranges can be modified after the cluster is provisioned. To do that, send a PATCH request with the new set of IP ranges:

```bash
   curl --request PATCH "https://$BROKER_URL/oauth/v2/service_instances/$INSTANCE_ID?accepts_incomplete=true" \
   --header 'X-Broker-API-Version: 2.14' \
   --header 'Content-Type: application/json' \
   --header "$AUTHORIZATION_HEADER" \
   --header 'Content-Type: application/json' \
   --data-raw "{
       \"service_id\": \"47c9dcbf-ff30-448e-ab36-d3bad66ba281\",
       \"plan_id\": \"4deee563-e5ec-4731-b9b1-53b42d855f0c\",
       \"context\": {
           \"globalaccount_id\": \"$GLOBAL_ACCOUNT_ID\"
       },
       \"parameters\": {
           \"accessControlList\": {
               \"allowedCIDRs\": [\"1.5.3.0/24\"]
           }
       }
   }"
```

If the `accessControlList` parameter is not provided, the cluster is created without any restrictions. It means that all IP addresses are allowed to access the Kubernetes API, but the user must be authorized to access it.
If the update request does not contain `accessControlList` parameter, the existing Access Control List is stay unchanged. To remove Access Control List, set `allowedCIDRs` to an empty list (`[]`):

```bash
   curl --request PATCH "https://$BROKER_URL/oauth/v2/service_instances/$INSTANCE_ID?accepts_incomplete=true" \
   --header 'X-Broker-API-Version: 2.14' \
   --header 'Content-Type: application/json' \
   --header "$AUTHORIZATION_HEADER" \
   --header 'Content-Type: application/json' \
   --data-raw "{
       \"service_id\": \"47c9dcbf-ff30-448e-ab36-d3bad66ba281\",
       \"plan_id\": \"4deee563-e5ec-4731-b9b1-53b42d855f0c\",
       \"context\": {
           \"globalaccount_id\": \"$GLOBAL_ACCOUNT_ID\"
       },
       \"parameters\": {
           \"accessControlList\": {
               \"allowedCIDRs\": []
           }
       }
   }"
```
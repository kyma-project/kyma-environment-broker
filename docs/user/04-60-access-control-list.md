<!--{"metadata":{"publish":false}}-->

# Access Control List

You can restrict access to the Kyma Kubernetes API using an Access Control List (ACL). Specify the IP ranges allowed to access the Kubernetes API. IPs that do not fall within any of the ranges are not allowed to access the API.
> [!NOTE]
> An unauthorized user can't access the API, even if their IP address falls within the specified range.

To define an Access Control List, provide the **accessControlList** parameter in the provisioning request. For example:


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

You can modify the set of IP ranges after the cluster is provisioned. To do that, send a PATCH request with the new set of IP ranges.

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

If you don't provide the **accessControlList** parameter or set **allowedCIDRs** to an empty list (`[]`), the cluster is created without any restrictions. It means that while all IP addresses can access the Kubernetes API, the user must be authorized to do that.
If the update request does not contain the **accessControlList** parameter, the existing Access Control List remains unchanged. To remove your Access Control List, set **allowedCIDRs** to an empty list (`[]`).

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
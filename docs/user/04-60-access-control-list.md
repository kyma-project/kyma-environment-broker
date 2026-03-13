# Access Control List

> ### Note:
> 
> The access control list is only available for Azure and AWS providers.

The access control list (ACL) feature allows to restrict access to Kubernetes API server. By default ACL is disabled for all plans.
To enable ACL, set the following property with plan names in the `kyma-environment-broker` configuration:

```yaml
  broker:
    ACLEnabledPlans:
      - azure
      - aws
```

See an example of the provisioning request with ACL:

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
           \"acl\": {
             \"allowedCIDRs\": [\"1.2.3.0/24\"]
           }
       }
   }"
```

The `acl` parameter is optional, you can skip this section in the provisioning request. In this case, the ACL is disabled and there are no restrictions on accessing the Kubernetes API server.
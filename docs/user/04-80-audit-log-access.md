<!--{"metadata":{"publish":false}}-->

# Audit Log Access

Kyma Environment Broker (KEB) allows you to enable Audit Log Access during SAP BTP, Kyma runtime provisioning and update operations.
By default, Audit Log Access is disabled.
To enable it, set the **auditLogAccess** parameter to `true` in the provisioning or update request.

> ### Note:
> The Audit Log Access feature is not available for `trial` Kyma instances or the `free` plan.

> ### Caution:
> Once you enable Audit Log Access, you cannot disable it.

## Provisioning with Audit Log Access

To provision a Kyma runtime with Audit Log Access enabled, add the **auditLogAccess** parameter set to `true` in the provisioning request.

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
           \"subaccount_id\": \"$SUBACCOUNT_ID\",
           \"user_id\": \"$USER_ID\"
       },
       \"parameters\": {
           \"name\": \"$NAME\",
           \"region\": \"$REGION\",
           \"auditLogAccess\": true
       }
   }"
```

## Updating Audit Log Access

To enable Audit Log Access on an existing Kyma runtime, send an update request with **auditLogAccess** set to `true`.

```bash
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
           \"user_id\": \"$USER_ID\"
       },
       \"parameters\": {
           \"auditLogAccess\": true
       }
   }"
```

> ### Note:
> If Audit Log Access is enabled, setting **auditLogAccess** to `false` in an update request results in an error.

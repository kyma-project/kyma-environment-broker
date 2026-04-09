# gVisor Container Runtime Sandbox

> ### Note:
> The gVisor container runtime sandbox feature is available only to allowlisted global accounts.
> Consequently, by default, gVisor is disabled.
> An attempt to enable gVisor for a non-allowlisted account results in an error.
> The **gvisor** parameter is validated against the global account allowlist on every field where it appears. If **gvisor** is set on any worker pool (main or additional) and the global account is not allowlisted, the entire request is rejected.

For cloud-native container security and portability, you can enable the [gVisor](https://gvisor.dev/) container runtime sandbox on the main Kyma worker pool and on additional worker node pools when provisioning or updating SAP BTP, Kyma runtime.

## Provisioning with gVisor

To provision a Kyma runtime with gVisor enabled on the main Kyma worker pool, add the **gvisor** parameter at the root level of the request with **enabled** set to `true`.

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
           \"subaccount_id\": \"$SUBACCOUNT_ID\",
           \"user_id\": \"$USER_ID\"
       },
       \"parameters\": {
           \"name\": \"$NAME\",
           \"region\": \"$REGION\",
           \"gvisor\": {
               \"enabled\": true
           }
       }
   }"
```

## Updating gVisor

To enable gVisor on an existing Kyma runtime, send an update request with the **gvisor** parameter.

```bash
   curl --request PATCH "https://$BROKER_URL/oauth/v2/service_instances/$INSTANCE_ID?accepts_incomplete=true" \
   --header 'X-Broker-API-Version: 2.14' \
   --header 'Content-Type: application/json' \
   --header "$AUTHORIZATION_HEADER" \
   --data-raw "{
       \"service_id\": \"47c9dcbf-ff30-448e-ab36-d3bad66ba281\",
       \"plan_id\": \"4deee563-e5ec-4731-b9b1-53b42d855f0c\",
       \"context\": {
           \"globalaccount_id\": \"$GLOBAL_ACCOUNT_ID\"
           \"subaccount_id\": \"$SUBACCOUNT_ID\",
           \"user_id\": \"$USER_ID\"
       },
       \"parameters\": {
           \"gvisor\": {
               \"enabled\": true
           }
       }
   }"
```

To disable gVisor, set **enabled** to `false`.

```bash
   curl --request PATCH "https://$BROKER_URL/oauth/v2/service_instances/$INSTANCE_ID?accepts_incomplete=true" \
   --header 'X-Broker-API-Version: 2.14' \
   --header 'Content-Type: application/json' \
   --header "$AUTHORIZATION_HEADER" \
   --data-raw "{
       \"service_id\": \"47c9dcbf-ff30-448e-ab36-d3bad66ba281\",
       \"plan_id\": \"4deee563-e5ec-4731-b9b1-53b42d855f0c\",
       \"context\": {
           \"globalaccount_id\": \"$GLOBAL_ACCOUNT_ID\"
           \"subaccount_id\": \"$SUBACCOUNT_ID\",
           \"user_id\": \"$USER_ID\"
       },
       \"parameters\": {
           \"gvisor\": {
               \"enabled\": false
           }
       }
   }"
```

If you omit the **gvisor** parameter from an update request, the existing gVisor configuration remains unchanged.

## gVisor on Additional Worker Node Pools

You can enable gVisor independently on each additional worker node pool by setting the **gvisor** parameter on individual items in the **additionalWorkerNodePools** list. The **gvisor** setting on the main worker pool and on additional worker node pools are independent of each other.

To provision a Kyma runtime with gVisor enabled on an additional worker node pool, include the **gvisor** parameter in the pool definition.

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
           \"subaccount_id\": \"$SUBACCOUNT_ID\",
           \"user_id\": \"$USER_ID\"
       },
       \"parameters\": {
           \"name\": \"$NAME\",
           \"region\": \"$REGION\",
           \"additionalWorkerNodePools\": [
               {
                   \"name\": \"worker-1\",
                   \"machineType\": \"Standard_D2s_v5\",
                   \"haZones\": true,
                   \"autoScalerMin\": 3,
                   \"autoScalerMax\": 20,
                   \"gvisor\": {
                       \"enabled\": true
                   }
               }
           ]
       }
   }"
```

You can also combine gVisor on the main worker pool and additional worker node pools in the same request.

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
           \"subaccount_id\": \"$SUBACCOUNT_ID\",
           \"user_id\": \"$USER_ID\"
       },
       \"parameters\": {
           \"name\": \"$NAME\",
           \"region\": \"$REGION\",
           \"gvisor\": {
               \"enabled\": true
           },
           \"additionalWorkerNodePools\": [
               {
                   \"name\": \"worker-1\",
                   \"machineType\": \"Standard_D2s_v5\",
                   \"haZones\": true,
                   \"autoScalerMin\": 3,
                   \"autoScalerMax\": 20,
                   \"gvisor\": {
                       \"enabled\": true
                   }
               }
           ]
       }
   }"
```

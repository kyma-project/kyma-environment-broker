# Custom Networking Configuration

To create a Kyma runtime with a custom IP range for worker nodes, specify the additional **networking** provisioning parameters. See the example:

```bash
   export VERSION=1.15.0
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
           \"networking\": {
              \"nodes\": \"10.250.0.0/20\"
           }
       }
   }"
```

> [!NOTE]
> The **networking** section is optional but if you use it, the **nodes** value is a mandatory field.

If you do not provide the **networking** object in the provisioning request, the default configuration is used.
The configuration is immutable - it cannot be changed later in an update request.
The provided IP range must not overlap with ranges of potential seed clusters (see [GardenerSeedCIDRs definition](https://github.com/kyma-project/kyma-environment-broker/blob/main/internal/networking/cidr.go)).
The suffix must not be greater than 23 because the IP range is divided between the zones and Nodes. Additionally, two ranges are reserved for `pods` and `services`, which, too, must not overlap with the IP range for Nodes.

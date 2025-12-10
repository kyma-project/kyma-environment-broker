# Custom Networking Configuration

With the **networking** section in the provisioning parameters, you can customize the following networking aspects of your Kyma runtime:

- Custom IP ranges for worker nodes, Pods, and services
- Dual-stack networking to enable both IPv4 and IPv6 protocols

> ### Note:
> All networking configurations are immutable and cannot be changed after provisioning.

## Custom IP Ranges

You can specify custom CIDR ranges for different network components. If you don't provide the **networking** object in the provisioning request, the default ranges are used.

### Worker Node IP Range

To create a Kyma runtime with a custom IP range for worker nodes, specify the **nodes** parameter.

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

### Complete Custom Networking

You can also specify custom ranges for Pods and services alongside worker nodes.

```bash
   \"networking\": {
      \"nodes\": \"10.250.0.0/20\",
      \"pods\": \"10.96.0.0/13\",
      \"services\": \"10.104.0.0/13\"
   }
```

> ### Note:
> - The provided IP range must not overlap with ranges of potential seed clusters (see [GardenerSeedCIDRs definition](https://github.com/kyma-project/kyma-environment-broker/blob/main/internal/networking/cidr.go)).
> - The suffix must not be greater than 23 because the IP range is divided between the zones and nodes. Additionally, two ranges are reserved for `pods` and `services`, which, too, must not overlap with the IP range for nodes.

## Dual-Stack Networking

Enable dual-stack networking to allow your Kyma runtime to support both IPv4 and IPv6 protocols simultaneously.

> ### Note:
> Dual-stack networking is only available for supported cloud providers: Amazon Web Services and Google Cloud.

### Dual-Stack Configuration

To enable dual-stack networking, you must include the **dualStack** parameter along with the mandatory **nodes** parameter.

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
              \"dualStack\": true,
              \"nodes\": \"10.250.0.0/20\"
           }
       }
   }"
```

### Combined Configuration

You can combine dual-stack networking with custom IP ranges.

```bash
   \"networking\": {
      \"dualStack\": true,
      \"nodes\": \"10.250.0.0/20\",
      \"pods\": \"10.96.0.0/13\",
      \"services\": \"10.104.0.0/13\"
   }
```

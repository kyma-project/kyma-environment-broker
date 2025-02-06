# Additional Worker Node Pools

To create an SAP BTP, Kyma runtime with additional worker node pools, specify the `additionalWorkerNodePools` provisioning parameter.

> [!NOTE]
> **name**, **machineType**, **haZones**, **autoScalerMin**, and **autoScalerMax** values are mandatory for additional worker node pool configuration.

See the example:

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
           \"additionalWorkerNodePools\": [
               {
                   \"name\": \"worker-1\",
                   \"machineType\": \"Standard_D2s_v5\",
                   \"haZones\": true,
                   \"autoScalerMin\": 3,
                   \"autoScalerMax\": 20
               },
               {
                   \"name\": \"worker-2\",
                   \"machineType\": \"Standard_D4s_v5\",
                   \"haZones\": false,
                   \"autoScalerMin\": 1,
                   \"autoScalerMax\": 1
               }
           ]
       }
   }"
```

If you do not provide the `additionalWorkerNodePools` list in the provisioning request, no additional worker node pools are created.

The **haZones** property specifies whether high availability zones are supported. This setting is permanent and cannot be changed later. 
If high availability is disabled, all resources are placed in a single, randomly selected zone. In this case, you can set both **autoScalerMin** and **autoScalerMax** to `1`, which helps reduce costs. 
It is not recommended for production environments. When enabled, resources are distributed across three zones to enhance fault tolerance. 
Enabled HA requires setting **autoScalerMin** to the minimal value 3.

If you do not provide the `additionalWorkerNodePools` list in the update request, the saved additional worker node pools stay untouched.
However, if you provide an empty list in the update request, all additional worker node pools are removed.
Renaming the additional worker node pool will result in the deletion of the existing pool and the creation of a new one.

See the following JSON example without the `additionalWorkerNodePools` list:

```json
{
  "service_id" : "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
  "plan_id" : "4deee563-e5ec-4731-b9b1-53b42d855f0c",
  "context" : {
    "globalaccount_id" : {GLOBAL_ACCOUNT_ID}
  },
  "parameters" : {
    "region": {REGION},
    "name" : {CLUSTER_NAME}
  }
}
```

See the following JSON example, where the `additionalWorkerNodePools` is an empty list:

```json
{
   "service_id" : "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
   "plan_id" : "4deee563-e5ec-4731-b9b1-53b42d855f0c",
   "context" : {
      "globalaccount_id" : {GLOBAL_ACCOUNT_ID}
   },
   "parameters" : {
      "region": {REGION},
      "name" : {CLUSTER_NAME},
      "additionalWorkerNodePools": []
   }
}
```

To update additional worker node pools, provide a list of objects with values for the mandatory properties. Without these values, a validation error occurs.
The update operation overwrites the additional worker node pools with the list provided in the JSON file. See the following scenario:

1. An existing instance has the following additional worker node pools:

```json
{
  "additionalWorkerNodePools": [
    {
      "name": "worker-1",
      "machineType": "Standard_D2s_v5",
      "haZones": true,
      "autoScalerMin": 3,
      "autoScalerMax": 20
    },
    {
      "name": "worker-2",
      "machineType": "Standard_D4s_v5",
      "haZones": false,
      "autoScalerMin": 1,
      "autoScalerMax": 1
    }
  ]
}
```

2. A user sends an update request (HTTP PUT) with the following JSON file in the payload:
```json
{
  "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
  "plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
  "context": {
    "globalaccount_id" : {GLOBAL_ACCOUNT_ID}
  },
  "parameters": {
    "name" : {CLUSTER_NAME},
    "additionalWorkerNodePools": [
      {
        "name": "worker-3",
        "machineType": "Standard_D8s_v5",
        "haZones": true,
        "autoScalerMin": 10,
        "autoScalerMax": 30
      }
    ]
  }
}
```

3. The additional worker node pools are updated to include the values of the `additionalWorkerNodePools` list from JSON file provided in the update request:
```json
{
  "additionalWorkerNodePools": [
    {
      "name": "worker-3",
      "machineType": "Standard_D8s_v5",
      "haZones": true,
      "autoScalerMin": 10,
      "autoScalerMax": 30
    }
  ]
}
```

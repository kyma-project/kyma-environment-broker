# Additional Worker Node Pools

> [!NOTE]
> This feature is still being developed and will be available soon.

To create an SAP BTP, Kyma runtime with additional worker node pools, specify the `additionalWorkerNodePools` provisioning parameter.

> [!NOTE]
> **name**, **machineType**, **autoScalerMin**, and **autoScalerMax** values are mandatory for additional worker node pool configuration.

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
           \"additionalWorkerNodePools\": {
               \"list\": [
                   {
                       \"name\": \"worker-1\",
                       \"machineType\": \"Standard_D2s_v5\",
                       \"autoScalerMin\": 3,
                       \"autoScalerMax\": 20
                   },
                   {
                       \"name\": \"worker-2\",
                       \"machineType\": \"Standard_D4s_v5\",
                       \"autoScalerMin\": 5,
                       \"autoScalerMax\": 25
                   }
               ]
           }
       }
   }"
```

If you do not provide the `additionalWorkerNodePools` object in the provisioning request or set the **skipModification** property to `true`, no additional worker node pools are created.

If you do not provide the `additionalWorkerNodePools` object in the update request or set the **skipModification** property to `true`, the saved additional worker node pools stay untouched.
However, if you provide an empty list in the update request, all additional worker node pools are removed.

See the following JSON example without the `additionalWorkerNodePools` object:

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

See the following JSON example with the `additionalWorkerNodePools` object with `skipModification` property set to `true`:

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
    "additionalWorkerNodePools" : {
      "skipModification": true
    }
  }
}
```

See the following JSON example, where the `additionalWorkerNodePools` object contains an empty list:

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
      "additionalWorkerNodePools" : {
         "list": []
      }
   }
}
```

To update additional worker node pools, provide a list of objects with values for the mandatory properties. Without these values, a validation error occurs.
The update operation overwrites the additional worker node pools with the list provided in the JSON file. See the following scenario:

1. An existing instance has the following additional worker node pools:

```json
{
  "additionalWorkerNodePools": {
    "list": [
      {
        "name": "worker-1",
        "machineType": "Standard_D2s_v5",
        "autoScalerMin": 3,
        "autoScalerMax": 20
      },
      {
        "name": "worker-2",
        "machineType": "Standard_D4s_v5",
        "autoScalerMin": 5,
        "autoScalerMax": 25
      }
    ]
  }
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
    "additionalWorkerNodePools": {
      "list": [
        {
          "name": "worker-3",
          "machineType": "Standard_D8s_v5",
          "autoScalerMin": 10,
          "autoScalerMax": 30
        }
      ]
    }
  }
}
```

3. The additional worker node pools are updated to include the values of the `additionalWorkerNodePools` object from JSON file provided in the update request:
```json
{
   "additionalWorkerNodePools": {
      "list": [
         {
            "name": "worker-3",
            "machineType": "Standard_D8s_v5",
            "autoScalerMin": 10,
            "autoScalerMax": 30
         }
      ]
   }
}
```

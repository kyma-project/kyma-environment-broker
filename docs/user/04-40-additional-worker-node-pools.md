# Additional Worker Node Pools

To create an SAP BTP, Kyma runtime with additional worker node pools, specify the `additionalWorkerNodePools` provisioning parameter.
To use the additional worker node pool feature, you must provide the following values: **name**, **machineType**, **haZones**, **autoScalerMin**, and **autoScalerMax**.

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

If you do not provide the `additionalWorkerNodePools` list in the update request, the saved additional worker node pools stay unchanged.
However, if you provide an empty list in the update request, all additional worker node pools are removed.
If you rename your existing additional worker node pool, it is deleted, and a new one is created.

The **haZones** property specifies whether high availability zones are supported. This setting is permanent and cannot be changed later. 

With high availability enabled, resources are distributed across three zones to enhance fault tolerance.
In this scenario, you must set **autoScalerMin** to at least `3`.

If high availability is disabled, all resources are placed in a single, randomly selected zone. In this case, you can set **autoScalerMin** to `0` and **autoScalerMax** to `1`, which helps reduce costs. 
However, it is not recommended for production environments. Setting the **autoScalerMin** value to 0 results in an automatic removal of the additional worker nodes depending on the current workload in the cluster.
If there are no user workloads deployed onto the nodes associated with the given additional worker node pool, the nodes are removed automatically. The process should take around 30 minutes.

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
          "autoScalerMin": 0,
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

3. The additional worker node pools are updated to include the values of the `additionalWorkerNodePools` list from the JSON file provided in the update request:

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

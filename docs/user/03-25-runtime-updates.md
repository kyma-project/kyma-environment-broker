# SAP BTP, Kyma Runtime Updates

## Overview

According to [OSB API specification](https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#updating-a-service-instance), a Kyma runtime update request can be processed synchronously or asynchronously. The asynchronous process is the default, and it is triggered when the update request contains parameter changes.
Synchronous processing can occur when there is no need to run an updating operation. This optimization prevents the creation and processing of multiple operations.

## Configuration

### Synchronous Processing

Synchronous update processing, which does not require an operation, is disabled by default. To enable this feature, set the following configuration in KEB:
```yaml
  broker:
    syncEmptyUpdateResponseEnabled: true
```

## Identical Updates

If an update request does not modify any runtime parameters and the last operation succeeded, Kyma Environment Broker does not need to perform any action and can respond synchronously with the HTTP `200` status code. For example:
The instance has been provisioned using the following request:
   ```bash
   curl --request PUT \
   --url http://localhost:8080/oauth/v2/service_instances/azure-cluster \
   --header 'Content-Type: application/json' \
   --header 'X-Broker-API-Version: 2.16' \
   --data '{
      "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
      "plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
      "context": {
         "globalaccount_id": "2f5011af-2fd3-44ba-ac60-eeb1148c2995",
         "subaccount_id": "8b9a0db4-9aef-4da2-a856-61a4420b66fd",
         "user_id": "user@email.com",
         "sm_operator_credentials": {
            "clientid": "cid",
            "clientsecret": "cs",
            "url": "url",
            "sm_url": "sm_url"
         }
      },
      "parameters": {
         "name": "azure-cluster",
         "region": "northeurope"
      }
   }'
   ```
Then, an update is triggered:
   ```bash
   curl --request PATCH \
   --url http://localhost:8080/oauth/v2/service_instances/azure-cluster \
   --header 'Content-Type: application/json' \
   --header 'X-Broker-API-Version: 2.16' \
   --data '{
      "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
      "plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
      "context": {
      },
      "parameters": {
         "machineType": "Standard_D2s_v5"
      }
   }'
   ```
The broker returns the HTTP `202` status code because the **machineType** parameter has changed, and the update operation is created and processed asynchronously. Wait for the operation to finish.
The next update request does not modify any parameters:
   ```bash
   curl --request PATCH \
   --url http://localhost:8080/oauth/v2/service_instances/azure-cluster \
   --header 'Content-Type: application/json' \
   --header 'X-Broker-API-Version: 2.16' \
   --data '{
      "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
      "plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
      "context": {
      },
      "parameters": {
         "machineType": "Standard_D2s_v5"
      }
   }'
   ```
The broker returns the HTTP `200` status code because no update operation is needed. Nothing has changed.

Yet another update modifies the machine type again:
   ```bash
   curl --request PATCH \
   --url http://localhost:8080/oauth/v2/service_instances/azure-cluster \
   --header 'Content-Type: application/json' \
   --header 'X-Broker-API-Version: 2.16' \
   --data '{
      "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
      "plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
      "context": {
      },
      "parameters": {
         "machineType": "Standard_D4s_v5"
      }
   }'
   ```
The response is HTTP `202` because the **machineType** parameter has changed, and the update operation is created and processed asynchronously.
Do not wait for success. Execute the next update request that does not modify any parameters:
   ```bash
   curl --request PATCH \
   --url http://localhost:8080/oauth/v2/service_instances/azure-cluster \
   --header 'Content-Type: application/json' \
   --header 'X-Broker-API-Version: 2.16' \
   --data '{
      "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
      "plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
      "context": {
      },
      "parameters": {
         "machineType": "Standard_D4s_v5"
      }
   }'
   ```
You see the same response with the same operation ID as the previous update request, because the last operation has not finished, and the parameters are the same. The broker returns the HTTP `202` status code, but no new operation is created.

## Last Operation Has Not Finished

The update request is processed asynchronously until the last operation finishes. The update request starts a new operation when the last operation failed, because the runtime may be in an unexpected state. The update operation is a way to verify the runtime status and provide that information to the user. 
If the last operation is still in progress but the parameters are the same, you get the HTTP `202 Accepted` status, but no new operation is created. The response contains the operation ID of the last operation.

## Suspension and Unsuspension

The **active** context parameter change is always processed synchronously - the response status is HTTP `200` even if, under the hood, a new operation is created.
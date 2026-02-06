# SAP BTP, Kyma Runtime updates

## Overview

According to [OSB API specification](https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#updating-a-service-instance), Kyma Runtime update reqeust could be processed synchronously or asynchronously. The asynchronous process is the default one, and it is triggered when the update request contains changes in parameters.
The synchronous processing could happen, when there is no need to run updating operation. This optimization prevents from creating and processing multiple operations.

## Identical updates

If an update request does not modify any parameters of the runtime and the last operation has succeeded, Kyma Environment Broker does not need to perform any action and could response synchronously with HTTP 200 status code. For example:
The instance is being provisioned using the following request:
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
Then an update is triggered:
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
The second update request which does not modify any parameter:
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
The broker response with HTTP 200 status, because there is no need to create an update operation. Nothing has changed.

# Last operation has not finished

The update request is processed asynchronously when the last operation has not finished. The update request start a new operation, when:
 - the last operation has failed, because the runtime may be in an unexpected state and the update operation is a way to verify the runtime status and provide those information to the user.
 - the last operation is still in progress, because the last operation may fail and the broker could not guarantee that the runtime will be in the expected state.

## Suspension and unsuspension

The `active` context parameter change is palways rocessed synchronously - the response status is HTTP 200 even if under the hood a new operation is created.
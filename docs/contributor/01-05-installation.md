# Install the Kyma Environment Broker

## Prerequisites

* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
* Kubernetes cluster, or [k3d](https://k3d.io) for local installation
* [yq](https://github.com/mikefarah/yq)

## Procedure

1. To install Kyma Environment Broker, use one of the following commands:

    ```bash
    make install
    ```

    ```bash
    make install version=1.18.0
    ```

    ```bash
    make install version=PR-1980
    ```

2. To provision an instance, use the following command:

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
         "user_id": "user@email.com"
      },
      "parameters": {
         "name": "azure-cluster",
         "region": "northeurope"
      }
   }'
   ```

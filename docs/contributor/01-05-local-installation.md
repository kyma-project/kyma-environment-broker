<!--{"metadata":{"publish":false}}-->

# Install Kyma Environment Broker Locally

## Local Architecture

![Local KEB architecture](../assets/local-keb-arch.drawio.svg)

The diagram illustrates the architecture based on an example provisioning request.
1. The user sends a provisioning request.
2. Kyma Environment Broker (KEB) requests an appropriate SecretBinding (CredentialsBinding).
3. KEB creates a Runtime resource.
4. KEB creates a Kyma resource.

## Prerequisites

* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
* A Kubernetes cluster, or [k3d](https://k3d.io) for local installation
* [yq](https://github.com/mikefarah/yq)

## Procedure

1. To install KEB, use one of the following commands:

    ```bash
    make install
    ```

    ```bash
    make install VERSION=1.18.0
    ```

    ```bash
    make install VERSION=PR-1980
    ```
   
2. Before performing any operation on the instance, port forward the necessary services to your local machine:

   ```bash
   kubectl port-forward -n kcp-system deployment/kcp-kyma-environment-broker 8080:8080 5432:5432
   ```

3. To provision the instance, use the following command. Note that this is an example of a command you can use for instance provisioning.

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

4. To simulate the kubeconfig created by Kyma Infrastructure Manager (KIM), create a kubeconfig Secret for the runtime.

   ```bash
   make create-kubeconfig-secret RUNTIME_ID=<runtime_id>
   ```

5. To simulate KIM and transition the runtime to the `Ready` state, use the following command:

   ```bash
   make set-runtime-state RUNTIME_ID=<runtime_id> STATE=Ready
   ```

6. To simulate Kyma Lifecycle Manager (KLM) and transition the Kyma instance to the `Ready` state, use the following command:

   ```bash
   make set-kyma-state KYMA_ID=<kyma_id> STATE=Ready
   ```

7. To update the instance, use the following command. Note that this is an example of a command you can use for instance updating.

   ```bash
   curl --request PATCH \
   --url http://localhost:8080/oauth/v2/service_instances/azure-cluster?accepts_incomplete=true \
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
         "machineType": "Standard_D4s_v5"
      }
   }'
   ```

8. To deprovision the instance, use the following command. Note that this command can only work if you created the instance using the command provided in this document.

   ```bash
   curl --request DELETE \
   --url http://localhost:8080/oauth/v2/service_instances/azure-cluster?accepts_incomplete=true&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281&plan_id=4deee563-e5ec-4731-b9b1-53b42d855f0c \
   --header 'Content-Type: application/json' \
   --header 'X-Broker-API-Version: 2.16' \
   --data '{
      "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
      "plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
      "context": {
         "globalaccount_id": "2f5011af-2fd3-44ba-ac60-eeb1148c2995",
         "subaccount_id": "8b9a0db4-9aef-4da2-a856-61a4420b66fd",
         "user_id": "user@email.com"
      }
   }'
   ```

## Seeding Analytics Data Locally

To populate the local cluster with test data for the KEB Analytics dashboard, use the following workflow.

### Prerequisites

* `python3` with `psycopg2` and `requests` packages:

  ```bash
  pip install psycopg2-binary requests
  ```

### Procedure

1. If you have an existing local cluster, recreate it to start clean:

   ```bash
   ./scripts/recreate-local-cluster.sh
   ```

2. Install KEB:

   ```bash
   make install
   ```

3. Create dummy CredentialsBinding CRs in `garden-kyma-dev` to support seeding large numbers of instances without exhausting the HAP pool (the local `aws` limit is 3 instances per binding):

   ```bash
   ./scripts/create-local-hap-bindings.sh --instances 1000
   ```

4. Port-forward the KEB service and the database:

   ```bash
   kubectl port-forward -n kcp-system deployment/kcp-kyma-environment-broker 8080:8080 5432:5432
   ```

5. Seed instances using the analytics seed script. Adjust `--instances` as needed:

   ```bash
   python3 utils/seed_analytics.py --count 1000
   ```

6. Spread the `created_at` timestamps over the past 90 days so that time-based charts show meaningful data:

   ```bash
   python3 utils/backdate_operations.py --days 90
   ```

7. Port-forward the analytics service:

   ```bash
   kubectl port-forward -n kcp-system deployment/kcp-kyma-environment-broker 8081:8081
   ```

8. Open `http://localhost:8081` to view the KEB Analytics dashboard. Use the **Refresh** button to reload aggregated data after seeding.

### Kyma Control Plane (KCP) CLI

To list, filter, and observe runtimes and their various attributes and states using the KCP CLI, set the **keb-api-url** to `http://localhost:8080` in the KCP configuration file.

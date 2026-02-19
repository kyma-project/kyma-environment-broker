# Kyma Binding Processes

## Creating a Kyma Binding

The binding creation process, which starts with a PUT HTTP request sent to the `/oauth/v2/service_instances/{{instance_id}}/service_bindings/{{binding_id}}` endpoint, produces a binding with a kubeconfig that contains a JWT token used for user authentication. The token is generated using Kubernetes TokenRequest attached to a ServiceAccount, ClusterRole, and ClusterRoleBinding, all named `kyma-binding-{{binding_id}}`. Such an approach allows for modifying the permissions granted with the kubeconfig.
Besides the kubeconfig, the response contains metadata with the **expires_at** field, which specifies the expiration time for the kubeconfig.
To explicitly specify the duration for which the generated kubeconfig is valid, provide the **expiration_seconds** in the `parameter` object of the request body.

The following diagram shows the flow of creating a service binding in Kyma Environment Broker (KEB). The process starts with a PUT request sent to KEB API.

> ### Note:
> On the diagram, "error" refers to a foreseen error in the process, not a server error.

![Bindings Create Flow](../assets/bindings-create-flow.drawio.svg)

The creation process is divided into three parts: configuration check, validation request, and binding creation.

### Configuration Check

If a feature flag for Kyma bindings is enabled, KEB performs the following steps:
1. Checks if the Kyma instance exists.
2. If the instance is found and the plan it has been provisioned with is bindable, KEB proceeds to the validation phase.

### Validation Request

1. The unmarshalled request is validated, and the correctness of its structure is checked. For the data that you can pass to the request, see the following table:

   | Name                   | Default | Description                            |
   |------------------------|---------|----------------------------------------|
   | **expiration_seconds** | `600`   | Specifies in seconds how long the generated kubeconfig is valid. The default, and at the same time the minimum possible value, is `600` seconds (10 minutes). The maximum possible value is `7200` seconds (2 hours). |

2. The first check verifies the expiration value. The minimum and maximum limits are configurable and, by default, set to 600 and 7200 seconds, respectively.
3. KEB checks the status of the instance. The instance must be provisioned for the binding creation.
4. KEB checks if the binding already exists. The binding in the database is identified by the Kyma instance ID and the binding ID, which are passed as a path query parameters. If the binding exists, KEB checks the values of the parameters of the existing binding. The OSB API requires that a request to create a binding fails if an object has already been created and the request contains different parameters.
5. If the found binding is not expired, KEB returns it in the response. If the found binding is expired and exists in the database, KEB responds with an error and a `Bad Request` status. This check is done in an implicit database insert statement. The query fails for expired but existing bindings because the primary key is defined on the instance and binding IDs, not the expiration date. This is the case until the cleanup job removes the expired binding from the database. If the binding does not exist, the flow returns to the process's execution path, where no bindings exist in the database.
6. Whether the binding exists or not, the last step in the request validation is to verify the number of bindings. Every instance is allowed to create a limited number of active bindings. The limit is configurable and, by default, set to 10 non-expired bindings. If the limit is not exceeded and the binding does not exist in the database, KEB proceeds to the next phase of the process: binding creation.

### Binding Creation

Binding creation consists of the following steps:
1. KEB creates a service binding object and generates a kubeconfig file with a JWT token. The kubeconfig file is valid for a specified period.

   > ### Note:
   > Expired bindings do not count towards the bindings limit. However, as long as they exist in the database, they prevent creating new bindings with the same ID. Only after they are removed by the cleanup job or manually can the binding be recreated.

2. KEB creates ServiceAccount, ClusterRole (administrator privileges), and ClusterRoleBinding, all named `kyma-binding-{{binding_id}}`. You can use the ClusterRole to modify permissions granted to the kubeconfig.
3. The created resources are used to generate a [TokenRequest](https://kubernetes.io/docs/reference/kubernetes-api/authentication-resources/token-request-v1/). The token is wrapped in a kubeconfig template and returned to the user.
4. The encrypted credentials are stored as an attribute in the previously created database binding.

   > ### Note:
   > It is not recommended to create multiple unused TokenRequest resources.

## Fetching a Kyma Binding

![Get Binding Flow](../assets/bindings-get-flow.drawio.svg)

The process starts with a GET request sent to the KEB API.
KEB checks if the Kyma instance exists. The found instance must not be deprovisioned or suspended. Otherwise, the endpoint doesn't return bindings for such an instance. 
Existing bindings are retrieved by instance ID and binding ID. If any bindings exist, they are filtered by expiration date. KEB returns only non-expired bindings.

## Deleting a Kyma Binding

![Delete Binding Flow](../assets/bindings-delete-flow.drawio.svg)

The process starts with a DELETE request sent to the KEB API. The first instruction is to check if the Kyma instance that the request refers to exists.
Any bindings of non-existing instances are treated as orphaned and removed. The next step is to conditionally delete the binding's ClusterRole, ClusterRoleBinding, and ServiceAccount, given that the cluster has been provisioned and not marked for removal. In case of deprovisioning or suspension of the Kyma cluster, this is unnecessary because the cluster is removed anyway.
In case of errors during the resource removal, the binding database record should not be removed, which is why the resource removal happens before the binding database record removal.
Finally, the last step is to remove the binding record from the database.

> ### Caution:
> Do not remove the ServiceAccount because the removal invalidates all tokens generated for that account and thus revokes access to the cluster for all clients using the kubeconfig from the binding.

## Cleanup Job

The Cleanup Job is a separate process decoupled from KEB. It is a CronJob that cleans up expired or orphaned Kyma bindings from the database. The value of **expires_at** field in the binding database record determines whether a binding is expired. If the value is in the past, the binding is considered expired and is removed from the database. 

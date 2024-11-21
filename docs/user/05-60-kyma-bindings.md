# Kyma Bindings

The Kyma binding is an abstraction of Kyma Environment Broker (KEB) that allows generating credentials for accessing an SAP BTP, Kyma runtime instance created by KEB. The credentials are generated as an administrator kubeconfig file that you can use to access the Kyma runtime. They are wrapped in a service binding object, as is known in the Open Service Broker API (OSB API) specification. The generated kubeconfig contains a TokenRequest tied to its custom ServiceAccount, which allows for revoking permissions, restricting privileges using Kubernetes RBAC, and generating short-lived tokens.

![Kyma bindings components](../assets/bindings-general.drawio.svg)

KEB manages the bindings and keeps them in a database together with generated kubeconfigs stored in an encrypted format. Management of bindings is allowed through the KEB bindings API, which consists of three endpoints: PUT, GET, and DELETE. An additional cleanup job periodically removes expired binding records from the database.

You can manage credentials for accessing a given service through the bindings' HTTP endpoints. The API includes all subpaths of `v2/service_instances/<service_id>/service_bindings` and follows the OSB API specification. However, the requests are limited to PUT, GET, and DELETE methods. Bindings can be rotated by subsequent calls of a DELETE method for an old binding, and a PUT method for a new one. The implementation supports synchronous operations only. All requests are idempotent. Requests to create a binding are configured to time out after 15 minutes.

> [!NOTE]
> You can find all endpoints in the KEB [Swagger Documentation](https://kyma-env-broker.cp.stage.kyma.cloud.sap/#/Bindings).

## The Process of Creating a Kyma Binding

The binding creation process, which starts with a PUT HTTP request sent to the `/oauth/v2/service_instances/{{instance_id}}/service_bindings/{{binding_id}}` endpoint, produces a binding with a kubeconfig that encapsulates a JWT token used for user authentication. The token is generated using Kubernetes TokenRequest attached to a ServiceAccount, ClusterRole, and ClusterRoleBinding, all named `kyma-binding-{{binding_id}}`. Such an approach allows for modifying permissions granted with the kubeconfig.
Besides the kubeconfig, the response contains metadata with the **expires_at** field, which specifies the expiration time of the kubeconfig. 
To specify the duration for which the generated kubeconfig is valid explicitly, provide the **expiration_seconds** in the `parameter` object of the request body.

The following diagram shows the flow of creating a service binding in Kyma Environment Broker. The process starts with a PUT request sent to KEB API. 

> [!NOTE] 
> On the diagram, "error" refers to a foreseen error in the process, not a server error.

![Bindings Create Flow](../assets/bindings-create-flow.drawio.svg)

The creation process is divided into three parts: configuration check, request validation, and binding creation.

### Configuration Check

If a feature flag for Kyma bindings is enabled, KEB first checks if the Kyma instance exists. If the instance is found and the plan it has been provisioned with is bindable, KEB proceeds to the validation phase.

### Request Validation

Now, the unmarshalled request is validated, and the correctness of its structure is checked. See the table for the data that you can pass to the request:

| Name                   | Default | Description                                                                                                                                                                                                                                                                                                                                                          |
|------------------------|---------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **expiration_seconds** | `600`   | Specifies the duration (in seconds) for which the generated kubeconfig is valid. If not provided, the default value of `600` seconds (10 minutes) is used, which is also the minimum value that can be set. The maximum value that can be set is `7200` seconds (2 hours).                                             |

After unmarshaling, the data is validated against the allowed parameter values, which includes checks of the expiration value range, database record existence, parameter mutations verification, expiration of existing bindings, and binding number limits. 
The first check verifies the expiration value. The minimum and maximum limits are configurable and, by default, set to 600 and 7200 seconds, respectively. 
Next, KEB checks if the binding already exists. The binding in the database is identified by the Kyma instance ID and the binding ID, which are passed as a path query parameter. If the binding exists, KEB checks the mutation of the parameters of the existing binding. 
The OSB API requires a request to create a binding to fail if an object has already been created and the request contains different parameters. 
Next, if the found binding is not expired, KEB returns it in the response. At this point, the flow returns to the process's execution path, where no bindings exist in the database. 
Whether the binding exists or not, the last step in the request validation is to verify the number of bindings. 
Every instance is allowed to create a limited number of bindings. The limit is configurable and, by default, set to 10. 
If the limit is not exceeded, KEB proceeds to the next phase of the process - binding creation.   


### Binding Creation

In the binding creation phase, KEB creates a service binding object and generates a kubeconfig file with a JWT token. The kubeconfig file is valid for a specified period, defaulted or set in the request body. The first step in this part is to check again if an expired binding exists in the database. 
This check is done in an implicit database insert statement. The query fails for expired but existing bindings because the primary key is defined on the instance and binding IDs, not the expiration date. This is the case until the expired binding is removed from the database by the cleanup job.  

> [!NOTE]
>  Expired bindings do not count towards the bindings limit. However, they prevent creating new bindings until they exist in the database. Only after they are removed by the cleanup job or manually can the binding be recreated again.

After the insert has been done, KEB creates ServiceAccount, ClusterRole (administrator privileges), and ClusterRoleBinding, all named `kyma-binding-{{binding_id}}`. You can use the ClusterRole to modify permissions granted to the kubeconfig.

The created resources are then used to generate a [TokenRequest](https://kubernetes.io/docs/reference/kubernetes-api/authentication-resources/token-request-v1/). The token is then wrapped in a kubeconfig template and returned to the user. The encrypted credentials are then stored as an attribute in the previously created database binding.

> [!NOTE]
>  Creation of multiple and unused TokenRequest resources is not recommended.


## The Process of Fetching a Kyma Binding

![Get Binding Flow](../assets/bindings-get-flow.drawio.svg)

The diagram shows a flow of fetching a Kyma binding in KEB. The process starts with a GET request sent to the KEB API. 
The first instruction in the process is to check if a Kyma instance exists. If a Kyma instance exists, it must not be deprovisioned or suspended. 
The endpoint doesn't return bindings for such instances. Existing bindings are retrieved by instance ID and binding ID. If any bindings exist, they are filtered by expiration date. KEB returns only non-expired bindings.

## The Process of Deleting a Kyma Binding

![Delete Binding Flow](../assets/bindings-delete-flow.drawio.svg)

The diagram shows the flow of removing a Kyma binding. The process starts with a DELETE request sent to the KEB API. The first instruction is to check if the Kyma instance that the request refers to exists. 
Any bindings of non-existing instances are treated as orphaned and are removed. The next step is to conditionally delete the binding's ClusterRole, ClusterRoleBinding, and ServiceAccount, given that the cluster has been provisioned and not marked for removal. In case of deprovisioning or suspension of the Kyma cluster, this is unnecessary because the cluster is removed either way. 
In case of errors during the resource removal process, the binding database record should not be removed, which is why the resource removal happens before the binding database record removal. 
Finally, the last step is to remove the binding record from the database. 

> [!WARNING]
> Removing the ServiceAccount invalidates all tokens generated for that account, revoking access to the cluster for all clients using the kubeconfig from the binding.

## Cleanup Job

The Cleanup Job is a separate process decoupled from KEB. It is a CronJob that cleans up expired or orphaned Kyma bindings from the database. The value of **expires_at** field in the binding database record determines whether a binding is expired. If the value is in the past, the binding is considered expired and is removed from the database. 

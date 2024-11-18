# Kyma Bindings

Kyma Binding is an abstraction of Kyma Environment Broker (KEB) that allows to generate credentials for accessing a SAP Kyma Runtime (SKR) created by KEB. The credentials are generated in the form of an admin kubeconfig file, that you can use to access the SKR, and are wrapped in a service binding object as it is known in the Open Service Broker API specification. The generated kubeconfig contains a TokenRequest that is tied to its custom ServiceAccount, which allows for revoking permissions, restricting privilleges using Kubernetes RBAC and short lived tokens generation.

![Kyma bindings components](../assets/bindings-general.drawio.svg)

KEB manages the bindings and keeps them in a database together with generated kubeconfigs stored in an encrypted format. Management of bindings is allowed through the KEB bindings API, which consists of three endpoints: PUT, GET, and DELETE. An additional cleanup job periodically removes expired binding records from the database.

## Request Examples

You can manage credentials for accessing a given service through the bindings' HTTP endpoints. The API includes all subpaths of `v2/service_instances/<service_id>/service_bindings` and follows the OSB API specification. However, the requests are limited to PUT, GET, and DELETE methods. Bindings can be rotated by subsequent calls of a DELETE method for an old binding, and a PUT method for a new one. The implementation supports synchronous operations only. All requests are idempotent. The `create binding` requests are configured to time out after 15 minutes.

> [!NOTE]
> You can find all endpoints in the KEB [Swagger Documentation](https://kyma-env-broker.cp.stage.kyma.cloud.sap/#/Bindings).

### Create a Service Binding

To create a binding, use a PUT request to KEB API:

```
PUT http://localhost:8080/oauth/v2/service_instances/{{instance_id}}/service_bindings/{{binding_id}}
Content-Type: application/json
X-Broker-API-Version: 2.14

{
  "service_id": "{{service_id}}",
  "plan_id": "{{plan_id}}",
  "parameters": {
    "expiration_seconds": 660
  }
}
```

If a binding is successfully created, the endpoint returns `201 Created` or `200 OK` status code depending on if the curret request created the binding or it already existed before.

### Fetching a Service Binding 

To fetch a binding, use a GET request to KEB API:

```
GET http://localhost:8080/oauth/v2/service_instances/{{instance_id}}/service_bindings/{{binding_id}}
X-Broker-API-Version: 2.14
```

KEB returns the `200 OK` status code with the kubeconfig in the response body. If the binding does not exist, the instance does not exist, or the instance is suspended, KEB returns a `404 Not Found` status code.

All the codes are based on the [Open Service Broker API specification](https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#fetching-a-service-binding) 

### Unbinding

To remove a binding, send a DELETE request to KEB API:

```
DELETE http://localhost:8080/oauth/v2/service_instances/{{instance_id}}/service_bindings/{{binding_id}}?plan_id={{plan_id}}&service_id={{service_id}}
X-Broker-API-Version: 2.14
```

If the binding is successfully removed, KEB returns the `200 OK` status code. If the binding or service instance does not exist, KEB returns the `410 Gone` code.

All the codes are based on the [Open Service Broker API specification](https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#unbinding) 

## Bindings Management

### Create Kyma Binding Process

The binding creation process, that starts with a PUT HTTP request sent to `/oauth/v2/service_instances/{{instance_id}}/service_bindings/{{binding_id}}` endpoint, produces a binding with a kubeconfig that encapsulates JWT token used for user authentication. The token is generated using Kubernetes TokenRequest attached to a ServiceAccount, ClusterRole, and ClusterRoleBinding, all named `kyma-binding-{{binding_id}}`. Such approach allows for modifying permissions granted with the kubeconfig.
Besides the kubeconfig, the response contains metadata with the **expires_at** field, which specifies the expiration time of the kubeconfig. 
To specify the duration for which the generated kubeconfig is valid explicitly, provide the **expiration_seconds** in the `parameter` object of the request body.


The diagram below shows the flow of creating a Service Binding in Kyma Environment Broker. The process starts with a PUT request sent to KEB API. 

![Bindings Create Flow](../assets/bindings-create-flow.drawio.svg)

> **NOTE**: On the diagram error means forseen error in the process, not a server error.

The creation process is devided into three parts: configuration check, request validation and binding creation.

<!-- Configuration Check -->
Given that a feature flag for Kyma Bindings is enabled, in the first instructions of the process KEB checks if the Kyma instance exists. If the instance is found and plan, that it has been provisioned with, is bindable, then KEB proceeds to validation phase.

<!-- Request Validation -->
It is now that the unmarshalled request is validated to check correctness of its structure. Allowed data that can be passed to the request includes:

| Name                   | Default | Description                                                                                                                                                                                                                                                                                                                                                          |
|------------------------|---------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **expiration_seconds** | `600`   | Specifies the duration (in seconds) for which the generated kubeconfig is valid. If not provided, the default value of `600` seconds (10 minutes) is used, which is also the minimum value that can be set. The maximum value that can be set is `7200` seconds (2 hours).                                             |

After unmarshaling the data is validated against allowed parameter values, which includes checks of expiration value range, database record existence, parameters mutations verification, expiration of exising bindings and binding number limits. The first of the checks is a verification of the expiration value. Minimum and maximum limits are configurable and, by default, set to 600 and 7200 seconds respectively. After that KEB checks if the binding already exists. The binding in the database is identified by Kyma instance ID and binding ID passed as a path query parameter. If the binding exists, KEB checks mutation of the parameters of the existing binding. The Open Service Broker API requires a create binding request to fail if an object has been already created and the request contains different parameters. Next, if the found binding is not expired, KEB returns it in the response. At this point, the flow gets back to the execution path of the process where no bindigs exist in database at all. Not matter if the binding exists or not the last step in the request validation is the verification of bindings number limit. Every instance is allowed to create a limited number of bindings. The limit is configurable and by default set to 10. If the bindings limit is not exceeded, KEB proceeds to the next phase of the process - binding creation.  


<!-- Binding Creation -->
In the binding creation phase, KEB creates a Service Binding object and generates a kubeconfig file with a JWT token. The kubeconfig file is valid for a specified time period, which is defaulted of set in the request body. The first step in this part is to check again if an expired binding exists in the database. This check is done implicitly in a database insert statement. The query will fail for expired but existing bindings because of primary key being defined on the instance and binding IDs and not expiration date. This will be the case until the expired binding is removed from the database by the cleanup job. 
> **Note:** Expired bindings do not count towards the bindings limit, however they will prevent from creating new bindings until they exist in the database. Only after they are removed by the cleanup job or manually, the binding can be recreated again.

After the insert has been done, KEB creates a ServiceAccount, ClusterRole (admin privileges), and ClusterRoleBinding, all named `kyma-binding-{{binding_id}}`. The ClusterRole can be used to modify permissions granted to the kubeconfig.

The created resources are then used to generate a [TokenRequest](https://kubernetes.io/docs/reference/kubernetes-api/authentication-resources/token-request-v1/). The token is then wrapped in a kubeconfig template and returned to the user. The encrypted credentials are then stored as an attribute in the previously created database binding.

> **NOTE**: Creation of multiple and unused TokenRequests is not recommended


### Fetching Kyma Binding Process

![Get Binding Flow](../assets/bindings-get-flow.drawio.svg)

The diagram shows a flow of fetching a Kyma binding in KEB. The process starts with a GET request sent to the KEB API. Bindings are located by instance and binding IDs. 
The first instruction in the process is to check if a Kyma instance exists. If a Kyma instance exists, it must not be deprovisioned or suspended. 
The endpoint doesn't return bindings for such instances. Existing bindings are loaded by instance ID and binding ID. If any bindings exist, they are filtered by expiration date. KEB returns only non-expired bindings.

### The  Process of Deleting a Kyma Binding

![Delete Binding Flow](../assets/bindings-delete-flow.drawio.svg)

The diagram shows the flow of removing a Kyma binding. The process starts with a DELETE request sent to the KEB API. The first instruction is to check if the Kyma instance that the request refers to exists. 
Any bindings of non-existing instances are treated as orphaned and are destined to be removed. The next step is to conditionally delete the binding's `ClusterRole`, `ClusterRoleBinding`, and `ServiceAccount`, given that the cluster has been provisioned and not marked for removal. In case of deprovisioning or suspension of the Kyma cluster, this is unnecessary because the cluster is removed either way. 
In case of errors during the resource removal process, the binding database record should not be removed, which is why the resource removal happens before the binding database record removal. 
Finally, the last step is to remove the binding record from the database. 

> [!WARNING]
> Removing the `ServiceAccount` invalidates all tokens generated for that account, revoking access to the cluster for all clients using the kubeconfig from the binding.

## Cleanup Job

The Cleanup Job is a separate process for cleanup of expired or orphaned Kyma bindings, decoupled from KEB. It is is a cronjob that removes expired binding records from the database. The expired binding is determined by the **expires_at** field in the binding database record. If the **expires_at** field is older than the current time, the binding is considered expired and is removed from the database. 

# Kyma Bindings

Kyma Binding is an abstraction in Kyma Environment Broker (KEB) that allows you to generate credentials for accessing a SAP Kyma Runtimes (SKR) created by KEB. The credentials are generated in the form of an admin kubeconfig file that you can use to access the SKR and are wrapped in Service Binding object as it is known in Open Service Broker API specification. The generated Kubeconfig contains a TokenRequest that is tied to its custom Service Account, which allows for revoking permissions, restricting privilleges using Kubernetes RBAC and short access kubeconfigs generation.

![Bindings Overview](../assets/bindings-general.drawio.svg)

The Bindings are managed by the KEB and kept in database together with generated Kubeconfigs stored in encrypted format. External manipulation is allowed through the Broker API consisting out of three endpoints: PUT, GET, and DELETE. As designated on the diagram there is a additional Cleanup Job that periodically removes expired binding records from database.

## API

You can manage credentials for accessing a given service through a Broker API endpoints related to bindings. The Broker API endpoints include all subpaths of `v2/service_instances/<service_id>/service_bindings` to serve that purpose. The endpoints follow the Open Service Broker API specification, however we are limiting its implementation to PUT, GET, and DELETE methods. Bindings rotation is supported through subsequent calls of DELETE for old binding and PUT for a new one. Implementation on Kyma side support synchronous operations only. All requests are idempotent. Additionally, there is a timeout of 15 minutes for the binding processes.

All the endpoints can be found in the KEB [Swagger Documentation](
https://kyma-env-broker.cp.stage.kyma.cloud.sap/#/Bindings)

### Creating a Service Binding

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

If binding is successfully create the endpoint returns `201 Created` or `200 OK` status code depending on if the request is the first one that created the binding or binding is already create at the time of processing the request.

### Fetching a Service Binding 

To fetch a binding, use a GET request to the Broker API:

```
GET http://localhost:8080/oauth/v2/service_instances/{{instance_id}}/service_bindings/{{binding_id}}
X-Broker-API-Version: 2.14
```

The Broker returns the `200 OK` status code with the kubeconfig in the response body. If the binding does not exist, the instance does not exist, or the instance is suspended, the Broker returns a `404 Not Found` status code.

All the codes are based on the [Open Service Broker API specification](https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#fetching-a-service-binding) 

### Unbinding

To remove a binding, send a DELETE request to the Broker API:

```
DELETE http://localhost:8080/oauth/v2/service_instances/{{instance_id}}/service_bindings/{{binding_id}}?plan_id={{plan_id}}&service_id={{service_id}}
X-Broker-API-Version: 2.14
```

If the binding is successfully removed, the Broker returns the `200 OK` status code. If the binding or service instance does not exist, the Broker returns the `410 Gone` code.

All the codes are based on the [Open Service Broker API specification](https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#unbinding) 

## Bindings Management

### Create Service Binding Process

The Broker returns a kubeconfig with a JWT token used as a user authentication mechanism. The token is generated using Kubernetes TokenRequest attached to a ServiceAccount, ClusterRole, and ClusterRoleBinding, all named `kyma-binding-{{binding_id}}`. Such an approach allows for modifying the permissions granted to the kubeconfig.
Besides the kubeconfig, there is metadata in the response with the **expires_at** field, which specifies the expiration time of the kubeconfig. 
To specify the duration for which the generated kubeconfig is valid, provide the **expiration_seconds** in the `parameter` object of the request body.

| Name                   | Default | Description                                                                                                                                                                                                                                                                                                                                                          |
|------------------------|---------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **expiration_seconds** | `600`   | Specifies the duration (in seconds) for which the generated kubeconfig is valid. If not provided, the default value of `600` seconds (10 minutes) is used, which is also the minimum value that can be set. The maximum value that can be set is `7200` seconds (2 hours).                                             |

Not Expired SB limits

The diagram below shows the flow of creating a Service Binding in Kyma Environment Broker. The process starts with a PUT request sent to the Broker API. 

![Bindings Create Flow](../assets/bindings-create-flow.drawio.svg)

On the diagram error means forseen error in the process, not a server error.

The creation process is devided into three parts: configuration check, request validation and binding creation.

<!-- Configuration Check -->
If the feature flag for Kyma Bindings is enabled on the KEB side, in the first instructions of the process the Broker checks if the Kyma Instance exists. If the Service Instance is found and plan that it has been provisioned in is of bindable plan then the Broker proceeds to binding specific validation phase.

<!-- Request Validation -->
Unmarshalled request is check for validity of parameters. Allowed data that can be passed to the request includes <insert table with possible data>. After unmarshaling the data is validated against allowed values which include expiration value range, database record existence and parameters mutations, expiration of exising bindings and binding limits. The first of the check is verification of expiration value. Minimum and maximum values are configurable and, by default, set to 600 and 7200 seconds respectively. After that the Broker checks if the binding already exists. The binding in database is identified by Kyma instance id and binding id passed as path query parameter. If the binding exists, that triggers check of values of parameters of the existing binding. Open Service Broker API requires the create binding request to fail if the object has been already created and current request contains different parameters. Next, if the found binding is not expired, the Broker returns it in the response. At this point the flow gets back to the execution path of the process used when no bindigs exist in database at all. Not matter if the binding exists or not that last step in request validation us verification of bindings limit. Every instance is allowed only to create a limited number of bindings. The limit is configurable and by default set to 10. If bindings limit is not exceeded the Broker proceeds to the next phase of the process - binding creation.  

<!-- Binding Creation -->
In the binding creation phase, the Broker creates a Service Binding object and generates a kubeconfig file with a JWT token. The kubeconfig file is valid for a specified duration, which is set in the request body. The Broker returns the kubeconfig contents in the response body. The first step in this part is to check again if expired binding exists in database. This check is introduced by implicit check in DB insert statement. The query will fail because of primary key being defined on instance id and binding id and not expiration date. This will be the case until the expired binding is remove from the database by the cleanup job. 
> **Note:** Expired bindings do not count towards the bindings limit, however they will prevent from creating new bindings until they exist in the database until they are removed by the cleanup job or manually removed using the unbind endpoint.
After the insert into database has been done the Broker creates a ServiceAccount, ClusterRole, and ClusterRoleBinding, all named `kyma-binding-{{binding_id}}`. Such an approach allows for modifying the permissions granted to the kubeconfig utilizing standard Kubernetes RBAC rules.

The created resources are then used to generate a [TokenRequest](https://kubernetes.io/docs/reference/kubernetes-api/authentication-resources/token-request-v1/) and put wrapped in a kubeconfig template to return ready to use credentials to the user. The credentials are stored as an attribute in the previously created database binding.

> **NOTE**: we do not recommend creation of multiple token requests so that they are not hanging without a purpose -->



### Fetching Service Binding Process

![Get Binding Flow](../assets/bindings-get-flow.drawio.svg)

The above diagram shows the flow of fetching a Service Binding in Kyma Environment Broker. The process starts with a GET request to the Broker API. Bindings are located by instance and binding ids.  The first instructions in the process if Kyma instance exists and if it exists, then it is not being deprovisioned or suspended. The endpoint will not return bindings for such instances. Existing Bindings are loaded byd instance id and binding id. If any bindings exists they are filter by expiration date. If the binding is not expired, the Broker returns only non expired bindings.

### Delete Service Binding Process

![Delete Binding Flow](../assets/bindings-delete-flow.drawio.svg)

The above diagram shows the flow of deleting a Service Binding in Kyma Environment Broker. The process starts with a DELETE request to the Broker API. The first instructions in the process is to check if Kyma instance that the request objects refer to exists. In this case, any bindings of non-existing instances are treated as orphaned and to be removed. The next step is to conditionally delete the binding's ClusterRole, ClusterRoleBinding and ServiceAccount given that the cluster is has been provisioned and is not marked for removal. In case of deprovisioning or suspension of Kyma cluster the an operation is not neccessary because either way cluster is marked for removal. In case of errors during this removal process needs binding record should not be removed which is why resource removal happens before the binding removal. Finally, the last step is to remove the binding record from the database. It is important to mention that this endpoint invalidates all tokens of a ServiceAccount and hence revokes access to the cluster for all clients using that binding.

## Cleanup Job


The Cleanup Job is a separate process from the binding creation process or KEB processes and runs independently. The idea behind is to keep binding removal process decoupled from KEB processes. It is is a cronjob that removes expired binding records from the database. 

The expiration time is determined by the **expires_at** field in the binding record. If the **expires_at** field is older than the current time, the binding is considered expired and is removed from the database. 

# Kyma Bindings

The Kyma binding is an abstraction of Kyma Environment Broker (KEB) that allows generating credentials for accessing an SAP BTP, Kyma runtime instance created by KEB. The credentials are generated as an administrator kubeconfig file that you can use to access the Kyma runtime. They are wrapped in a service binding object, as is known in the Open Service Broker API (OSB API) specification. The generated kubeconfig contains a TokenRequest tied to its custom ServiceAccount, which allows for revoking permissions, restricting privileges using Kubernetes RBAC, and generating short-lived tokens.

![Kyma bindings components](../assets/bindings-general.drawio.svg)

KEB manages the bindings and keeps them in a database together with generated kubeconfigs stored in an encrypted format. Management of bindings is allowed through the KEB bindings API, which consists of three endpoints: PUT, GET, and DELETE. An additional cleanup job periodically removes expired binding records from the database.

You can manage credentials for accessing a given service through the bindings' HTTP endpoints. The API includes all subpaths of `v2/service_instances/<service_id>/service_bindings` and follows the OSB API specification. However, the requests are limited to PUT, GET, and DELETE methods. Bindings can be rotated by subsequent calls of a DELETE method for an old binding, and a PUT method for a new one. The implementation supports synchronous operations only. All requests are idempotent. Requests to create a binding are configured to time out after 15 minutes.

> [!NOTE]
> You can find all endpoints in the KEB [Swagger Documentation](https://kyma-env-broker.cp.stage.kyma.cloud.sap/#/Bindings).

## Kyma Bindings API Request Examples

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

If a binding is successfully created, the endpoint returns one of the following responses: 
* `201 Created` if the current request created the binding. 
* `200 OK` if the binding already existed.

### Fetch a Service Binding

To fetch a binding, use a GET request to KEB API:

```
GET http://localhost:8080/oauth/v2/service_instances/{{instance_id}}/service_bindings/{{binding_id}}
X-Broker-API-Version: 2.14
```

KEB returns the `200 OK` status code with the kubeconfig in the response body.
The [cluster name](04-05-cluster-name.md) of the Kyma runtime is used as the context name in the generated kubeconfig file.
If the binding or the instance does not exist, or if the instance is suspended, KEB returns the `404 Not Found` status code.

All HTTP codes are based on the [OSB API specification](https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#fetching-a-service-binding).

### Remove a Service Binding

To remove a binding, send a DELETE request to KEB API:

```
DELETE http://localhost:8080/oauth/v2/service_instances/{{instance_id}}/service_bindings/{{binding_id}}?plan_id={{plan_id}}&service_id={{service_id}}
X-Broker-API-Version: 2.14
```

If the binding is successfully removed, KEB returns the `200 OK` status code. If the binding or service instance does not exist, KEB returns the `410 Gone` code.

All HTTP codes are based on the [OSB API specification](https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#unbinding).

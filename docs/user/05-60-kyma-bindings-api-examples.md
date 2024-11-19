# Kyma Bindings: Request Examples

## Create a Service Binding

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

If a binding is successfully created, the endpoint returns the `201 Created` if the current request created it or the `200 OK` status code if it already existed.

## Fetch a Service Binding 

To fetch a binding, use a GET request to KEB API:

```
GET http://localhost:8080/oauth/v2/service_instances/{{instance_id}}/service_bindings/{{binding_id}}
X-Broker-API-Version: 2.14
```

KEB returns the `200 OK` status code with the kubeconfig in the response body. If the binding or the instance does not exist, or if the instance is suspended, KEB returns the `404 Not Found` status code.

All HTTP codes are based on the [OSB API specification](https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#fetching-a-service-binding).

## Remove a Service Binding

To remove a binding, send a DELETE request to KEB API:

```
DELETE http://localhost:8080/oauth/v2/service_instances/{{instance_id}}/service_bindings/{{binding_id}}?plan_id={{plan_id}}&service_id={{service_id}}
X-Broker-API-Version: 2.14
```

If the binding is successfully removed, KEB returns the `200 OK` status code. If the binding or service instance does not exist, KEB returns the `410 Gone` code.

All HTTP codes are based on the [OSB API specification](https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#unbinding).

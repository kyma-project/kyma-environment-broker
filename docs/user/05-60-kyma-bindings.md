# Kyma Bindings

You can manage credentials for accessing a given service through a Broker API endpoint related to bindings. The Broker API endpoints include all subpaths of `v2/service_instances/<service_id>/service_bindings`. In the case of Kyma Environment Broker (KEB), the generated credentials are a kubeconfig for a managed Kyma cluster. To generate a kubeconfig for a given Kyma instance, send the following request to the Broker API:

```
PUT http://localhost:8080/oauth/v2/service_instances/{{instance_id}}/service_bindings/{{binding_id}}?accepts_incomplete=false&service_id={{service_id}}&plan_id={{plan_id}}
Content-Type: application/json
X-Broker-API-Version: 2.14

{
  "service_id": "{{service_id}}",
  "plan_id": "{{plan_id}}"
  "parameters": {
  }
}
```

The Broker returns a kubeconfig file in the response body. The kubeconfig file contains the necessary information to access the managed Kyma cluster:

| Name                   | Default | Description                                                                                                                                                                                                                                                                                                                                                          |
|------------------------|---------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **expiration_seconds** | `600`   | Specifies the duration (in seconds) for which the generated kubeconfig is valid. If not provided, the default value of `600` seconds (10 minutes) is used, which is also the minimum value that can be set. The maximum value that can be set is `7200` seconds (2 hours).                                             |

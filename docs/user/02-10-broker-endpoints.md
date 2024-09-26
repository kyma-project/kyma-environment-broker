# Kyma Environment Broker Endpoints

Kyma Environment Broker (KEB) implements the [Open Service Broker API (OSB API)](https://github.com/openservicebrokerapi/servicebroker/blob/v2.14/spec.md). All the OSB API endpoints are served with the following prefixes:

| Prefix | Description |
|---|---|
| `/oauth` | Defines a prefix for the endpoint secured with the OAuth2 authorization. The value for the SAP BTP region is specified under the **broker.defaultRequestRegion** parameter in the [`values.yaml`](https://github.com/kyma-project/kyma-environment-broker/blob/main/resources/keb/values.yaml) file. |
| `/oauth/{region}` | Defines a prefix for the endpoint secured with the OAuth2 authorization. The SAP BTP region value is specified in the request. |

> [!NOTE] 
> When the `{region}` value is one of EU Access BTP regions, the EU Access restrictions apply. For more information, see [EU Access](../contributor/03-20-eu-access.md).

Besides OSB API endpoints, KEB exposes the REST `/info/runtimes` endpoint that provides information about all created Runtimes, both succeeded and failed. This endpoint is secured with the OAuth2 authorization.

For more details on KEB APIs, see [`swagger`](../../resources/keb/files/swagger.yaml).

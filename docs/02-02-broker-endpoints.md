# Kyma Environment Broker endpoints

Kyma Environment Broker (KEB) implements the [Open Service Broker API](https://github.com/openservicebrokerapi/servicebroker/blob/v2.14/spec.md) (OSB API). All the OSB API endpoints are served with the following prefixes:

| Prefix | Description |
|---|---|
| `/oauth` | Defines a prefix for the endpoint secured with the OAuth2 authorization. The value for the SAP BTP region is specified under the **broker.defaultRequestRegion** parameter in the [`values.yaml`](https://github.com/kyma-project/control-plane/blob/main/resources/kcp/charts/kyma-environment-broker/values.yaml) file. |
| `/oauth/{region}` | Defines a prefix for the endpoint secured with the OAuth2 authorization. The SAP BTP region value is specified in the request. |

> **NOTE:** When the `{region}` value is one of EU Access BTP regions, the EU Access restrictions apply. For more information, see [EU Access](./03-18-eu-access.md).

Besides OSB API endpoints, KEB exposes the REST `/info/runtimes` endpoint that provides information about all created Runtimes, both succeeded and failed. This endpoint is secured with the OAuth2 authorization.

For more details on KEB APIs, see [this file](https://htmlpreview.github.io/?https://raw.githubusercontent.com/kyma-project/kyma-environment-broker/main/files/swagger/index.html).

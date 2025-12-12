# Authorization

Kyma Environment Broker (KEB) endpoints are secured by OAuth2 authorization. It is configured in the [authorization-policy](../../resources/keb/templates/authorization-policy.yaml) file.

To access the KEB Open Service Broker (OSB) endpoints, use the `/oauth` prefix before OSB API paths. For example:

```shell
/oauth/{region}/v2/catalog
```

You must also specify the `Authorization: Bearer` request header:

```shell
Authorization: Bearer {ACCESS_TOKEN}
```

## Get the Access Token

To obtain a new access token, run:

```shell
export CLIENT_ID={CLIENT_ID}
export CLIENT_SECRET={CLIENT_SECRET}
export TOKEN_URL={TOKEN_URL}
export ENCODED_CREDENTIALS=$(echo -n "$CLIENT_ID:$CLIENT_SECRET" | base64)

curl -ik -X POST $TOKEN_URL -H "Authorization: Basic $ENCODED_CREDENTIALS" --data "grant_type=client_credentials" --data "scope=broker:write"
```

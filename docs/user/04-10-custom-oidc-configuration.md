# Custom OIDC Configuration

To create an SAP BTP, Kyma runtime with a custom Open ID Connect (OIDC) configuration, you can specify either a single `oidc` object or a list of `oidc` objects as provisioning parameters. While both options are supported, using a list of `oidc` objects is the recommended approach, even if you are defining only one OIDC configuration. The list allows you to define multiple OIDC configurations. The single `oidc` object is only supported for backward compatibility.

> [!NOTE]
> `clientID` and `issuerURL` values are mandatory for custom OIDC configuration.

See the example with the OIDC list:

```bash
   export VERSION=1.15.0
   curl --request PUT "https://$BROKER_URL/oauth/v2/service_instances/$INSTANCE_ID?accepts_incomplete=true" \
   --header 'X-Broker-API-Version: 2.14' \
   --header 'Content-Type: application/json' \
   --header "$AUTHORIZATION_HEADER" \
   --header 'Content-Type: application/json' \
   --data-raw "{
       \"service_id\": \"47c9dcbf-ff30-448e-ab36-d3bad66ba281\",
       \"plan_id\": \"4deee563-e5ec-4731-b9b1-53b42d855f0c\",
       \"context\": {
           \"globalaccount_id\": \"$GLOBAL_ACCOUNT_ID\"
       },
       \"parameters\": {
           \"name\": \"$NAME\",
           \"region\": \"$REGION\",
           \"oidc\": {
              \"list\": [
                 {
                    \"clientID\": \"9bd05ed7-a930-44e6-8c79-e6defeb7dec5\",
                    \"issuerURL\": \"https://kymatest.accounts400.ondemand.com\",
                    \"groupsClaim\": \"groups\",
                    \"groupPrefix\": \"-\",
                    \"signingAlgs\": [\"RS256\"],
                    \"usernamePrefix\": \"-\",
                    \"usernameClaim\": \"sub\",
                    \"requiredClaims\": [],
                 }
              ]
           }
       }
   }"
```
<details>
<summary>See the example with the single OIDC object (not recommended):</summary>

```bash
   export VERSION=1.15.0
   curl --request PUT "https://$BROKER_URL/oauth/v2/service_instances/$INSTANCE_ID?accepts_incomplete=true" \
   --header 'X-Broker-API-Version: 2.14' \
   --header 'Content-Type: application/json' \
   --header "$AUTHORIZATION_HEADER" \
   --header 'Content-Type: application/json' \
   --data-raw "{
       \"service_id\": \"47c9dcbf-ff30-448e-ab36-d3bad66ba281\",
       \"plan_id\": \"4deee563-e5ec-4731-b9b1-53b42d855f0c\",
       \"context\": {
           \"globalaccount_id\": \"$GLOBAL_ACCOUNT_ID\"
       },
       \"parameters\": {
           \"name\": \"$NAME\",
           \"region\": \"$REGION\",
           \"oidc\": {
              \"clientID\": \"9bd05ed7-a930-44e6-8c79-e6defeb7dec5\",
              \"issuerURL\": \"https://kymatest.accounts400.ondemand.com\",
              \"groupsClaim\": \"groups\",
              \"signingAlgs\": [\"RS256\"],
              \"usernamePrefix\": \"-\",
              \"usernameClaim\": \"sub\"
           }
       }
   }"
```

</details>

If you do not include the `oidc` list or the single `oidc` object in the provisioning request, the default OIDC configuration is applied. However, if you provide an empty `oidc` list (with zero elements), no OIDC configuration will be applied to the instance. Unlike the single `oidc` object, which defaults to the predefined values when its properties are left empty, the `oidc` list does not inherit default values for its items and they need to be explicitly defined.

See the following JSON example without the `oidc` object or list:

```json
{
  "service_id" : "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
  "plan_id" : "4deee563-e5ec-4731-b9b1-53b42d855f0c",
  "context" : {
    "globalaccount_id" : {GLOBAL_ACCOUNT_ID}
  },
  "parameters" : {
    "region": {REGION},
    "name" : {CLUSTER_NAME}
  }
}
```

See the following JSON example with the `oidc` list whose item has empty properties:

```json
{
  "service_id" : "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
  "plan_id" : "4deee563-e5ec-4731-b9b1-53b42d855f0c",
  "context" : {
    "globalaccount_id" : {GLOBAL_ACCOUNT_ID}
  },
  "parameters" : {
    "region": {REGION},
    "name" : {CLUSTER_NAME},
    "oidc" : {
      "clientID" : "",
      "issuerURL" : "",
      "groupsClaim" : "",
      "groupsPrefix" : "",
      "signingAlgs" : [],
      "usernamePrefix" : "",
      "usernameClaim" : "",
      "requiredClaims" : []
    }
  }
}
```

This is the default OIDC configuration in JSON:

```json
{
  ...
    "oidc" : {
      "clientID" : "9bd05ed7-a930-44e6-8c79-e6defeb7dec9",
      "issuerURL" : "https://kymatest.accounts400.ondemand.com",
      "groupsClaim" : "groups",
      "groupsPrefix" : "-",
      "signingAlgs" : ["RS256"],
      "usernamePrefix" : "-",
      "usernameClaim" : "sub",
      "requiredClaims" : []
    }
  ...
}
```

See the following JSON example with the `oidc` object whose properties are empty:

```json
{
  "service_id" : "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
  "plan_id" : "4deee563-e5ec-4731-b9b1-53b42d855f0c",
  "context" : {
    "globalaccount_id" : {GLOBAL_ACCOUNT_ID}
  },
  "parameters" : {
    "region": {REGION},
    "name" : {CLUSTER_NAME},
    "oidc" : {
      "clientID" : "",
      "issuerURL" : "",
      "groupsClaim" : "",
      "groupsPrefix" : "",
      "signingAlgs" : [],
      "usernamePrefix" : "",
      "usernameClaim" : "",
      "requiredClaims" : []
    }
  }
}
```

This is the default OIDC configuration in JSON:

```json
{
  ...
    "oidc" : {
      "clientID" : "9bd05ed7-a930-44e6-8c79-e6defeb7dec9",
      "issuerURL" : "https://kymatest.accounts400.ondemand.com",
      "groupsClaim" : "groups",
      "groupsPrefix" : "-",
      "signingAlgs" : ["RS256"],
      "usernamePrefix" : "-",
      "usernameClaim" : "sub",
      "requiredClaims" : []
    }
  ...
}
```

To update the OIDC configuration, provide values for the mandatory properties. Without these values, a validation error occurs. If you omit the `oidc` list or the single `oidc` object in the update request, the existing OIDC configuration remains unchanged. Providing an empty `oidc` list clears the OIDC configuration for the instance. The update operation overwrites the OIDC configuration values provided in JSON, meaning that OIDC properties with empty values are considered valid and will replace the existing values. This behavior applies to both the `oidc` object and the `oidc` list.

### Scenario: Instance with an OIDC Object List

1. **Current OIDC Configuration**  
  The instance has the following OIDC object list configuration:

  ```json
  [
    {
     "ClientID": "9bd05ed7-a930-44e6-8c79-e6defeb7dec9",
     "IssuerURL": "https://kymatest.accounts400.ondemand.com",
     "GroupsClaim": "groups",
     "GroupsPrefix": "-",
     "UsernameClaim": "sub",
     "UsernamePrefix": "-",
     "SigningAlgs": ["RS256"],
     "RequiredClaims": []
    },
    {
     "ClientID": "3f2a1c8e-7b4d-4e2f-9a6b-2d8e5f7c9d12",
     "IssuerURL": "https://foo.bar.com",
     "GroupsClaim": "groups",
     "GroupsPrefix": "-",
     "UsernameClaim": "sub",
     "UsernamePrefix": "acme-",
     "SigningAlgs": ["RS256"],
     "RequiredClaims": []
    }
  ]
  ```

2. **Update Request**  
  The user sends an HTTP PATCH request with the following payload to update the OIDC configuration:

  ```json
  {
    "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
    "plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
    "context": {
     "globalaccount_id": "{GLOBAL_ACCOUNT_ID}"
    },
    "parameters": {
     "name": "{CLUSTER_NAME}",
     "oidc": {
      "list": [
        {
         "clientID": "9bd05ed7-a930-44e6-8c79-e6defeb7dec9",
         "issuerURL": "https://kymatest.accounts400.ondemand.com",
         "groupsClaim": "groups",
         "groupsPrefix": "-",
         "signingAlgs": ["RS256"],
         "usernameClaim": "sub",
         "usernamePrefix": "-",
         "requiredClaims": []
        },
        {
         "clientID": "test",
         "issuerURL": "https://test.com",
         "groupsClaim": "",
         "groupsPrefix": "",
         "signingAlgs": [],
         "usernameClaim": "",
         "usernamePrefix": "",
         "requiredClaims": []
        }
      ]
     }
    }
  }
  ```

3. **Updated OIDC Configuration**  
  After the update, the OIDC configuration is modified to reflect the values provided in the request:

  ```json
  [
    {
     "ClientID": "9bd05ed7-a930-44e6-8c79-e6defeb7dec9",
     "IssuerURL": "https://kymatest.accounts400.ondemand.com",
     "GroupsClaim": "groups",
     "GroupsPrefix": "-",
     "UsernameClaim": "sub",
     "UsernamePrefix": "-",
     "SigningAlgs": ["RS256"],
     "RequiredClaims": []
    },
    {
     "ClientID": "test",
     "IssuerURL": "https://test.com",
     "GroupsClaim": "",
     "GroupsPrefix": "",
     "UsernameClaim": "",
     "UsernamePrefix": "",
     "SigningAlgs": [],
     "RequiredClaims": []
    }
  ]
  ```


### Scenario: Instance with a Single OIDC Object

1. **Current OIDC Configuration**  
  The instance has the following OIDC object configuration:

  ```json
  {
    "clientID": "9bd05ed7-a930-44e6-8c79-e6defeb7dec9",
    "issuerURL": "https://kymatest.accounts400.ondemand.com",
    "groupsClaim": "groups",
    "groupsPrefix": "-",
    "usernameClaim": "sub",
    "usernamePrefix": "-",
    "signingAlgs": ["RS256"],
    "requiredClaims": []
  }
  ```

2. **Update Request**  
  The user sends an HTTP PATCH request with the following payload to update the OIDC configuration:

  ```json
  {
    "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
    "plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
    "context": {
     "globalaccount_id": "{GLOBAL_ACCOUNT_ID}"
    },
    "parameters": {
     "name": "{CLUSTER_NAME}",
     "oidc": {
      "clientID": "new-client-id",
      "issuerURL": "https://new-issuer-url.local.com",
      "groupsClaim": "",
      "groupsPrefix": "",
      "signingAlgs": [],
      "usernamePrefix": "",
      "usernameClaim": "",
      "requiredClaims": []
     }
    }
  }
  ```

3. **Updated OIDC Configuration**  
  After the update, the OIDC configuration is modified to reflect the values provided in the request:

  ```json
  {
    "clientID": "new-client-id",
    "issuerURL": "https://new-issuer-url.local.com",
    "groupsClaim": "",
    "groupsPrefix": "",
    "usernameClaim": "",
    "usernamePrefix": "",
    "signingAlgs": [],
    "requiredClaims": []
  }
  ```

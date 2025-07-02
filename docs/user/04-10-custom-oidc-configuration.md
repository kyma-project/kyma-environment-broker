# Custom OIDC Configuration

To create an SAP BTP, Kyma runtime with a custom Open ID Connect (OIDC) configuration, you can specify either a single `oidc` object or a list of `oidc` objects as provisioning parameters. While both options are supported, using a list of `oidc` objects is the recommended approach, even if you are defining only one OIDC configuration. The list allows you to define multiple OIDC configurations. The single `oidc` object is only supported for backward compatibility.

See the example with the OIDC list:

> [!NOTE]
> All fields except `requiredClaims` are mandatory when using the `oidc` list for custom OIDC configuration.

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
<summary>This solution is not recommended. It is only supported for backward compatibility with existing automations.</summary>

> [!NOTE]
> `clientID` and `issuerURL` values are mandatory when using the single `oidc` object for for custom OIDC configuration.

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

### Example 1: Without the `oidc` Object or List

This example demonstrates a request without specifying any `oidc` configuration. The default OIDC configuration is applied automatically.

**Request:**

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

**Applied OIDC Configuration:**

```json
{
  ...
  "oidc" : {
    "clientID" : "default-client-id",
    "issuerURL" : "https://default.issuer.com",
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

---

### Example 2: With the `oidc` List

This example shows a request with an `oidc` list containing a single configuration. The list allows defining multiple OIDC configurations.

**Request:**

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
      "list": [
        {
          "clientID" : "custom-client-id",
          "issuerURL" : "https://custom.issuer.com",
          "groupsClaim" : "groups",
          "groupsPrefix" : "-",
          "signingAlgs" : ["RS256"],
          "usernamePrefix" : "-",
          "usernameClaim" : "sub",
          "requiredClaims" : ["first-claim=value", "second-claim=value"]
        }
      ]
    }
  }
}
```

**Applied OIDC Configuration:**

```json
{
  ...
  "oidc" : {
    "list": [
      {
        "clientID" : "custom-client-id",
        "issuerURL" : "https://custom.issuer.com",
        "groupsClaim" : "groups",
        "groupsPrefix" : "-",
        "signingAlgs" : ["RS256"],
        "usernamePrefix" : "-",
        "usernameClaim" : "sub",
        "requiredClaims" : ["first-claim=value", "second-claim=value"]
      }
    ]
  }
  ...
}
```

---

### Example 3: With the `oidc` Object (Empty Properties)

This example illustrates a request with an `oidc` object where all properties are left empty. The default OIDC configuration is applied.

**Request:**

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

**Applied OIDC Configuration:**

```json
{
  ...
  "oidc" : {
    "clientID" : "default-client-id",
    "issuerURL" : "https://default.issuer.com",
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

## Updating the OIDC Configuration

To update the OIDC configuration, provide values for the mandatory properties. Without these values, a validation error occurs. If you omit the `oidc` list or the single `oidc` object in the update request, the existing OIDC configuration remains unchanged. Providing an empty `oidc` list clears the OIDC configuration for the instance. The update operation overwrites the OIDC configuration values provided in JSON for the `oidc` list, meaning that OIDC properties with empty values are considered valid and will replace the existing values. However, for the single `oidc` object, empty values do not change the configuration, and only the provided values are updated. It is possible to update the configuration from a single `oidc` object to an `oidc` list. However, updating from an `oidc` list to a single `oidc` object is not supported.

---

### Scenario 1: Updating an Instance with an OIDC Object List

1. **Current OIDC Configuration**  
  The instance has the following OIDC object list configuration:

  ```json
  [
    {
     "clientID": "first-custom-client-id",
     "issuerURL": "https://first.custom.issuer.com",
     "groupsClaim": "groups",
     "groupsPrefix": "-",
     "usernameClaim": "sub",
     "usernamePrefix": "-",
     "signingAlgs": ["RS256"],
     "requiredClaims": ["first-claim=value", "second-claim=value"]
    },
    {
     "clientID": "second-custom-client-id",
     "issuerURL": "https://second.custom.issuer.com",
     "groupsClaim": "groups",
     "groupsPrefix": "-",
     "usernameClaim": "sub",
     "usernamePrefix": "acme-",
     "signingAlgs": ["RS256"],
     "requiredClaims": ["example-claim=value"]
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
         "clientID": "new-client-id",
         "issuerURL": "https://new.issuer.com",
         "groupsClaim": "groups",
         "groupsPrefix": "-",
         "signingAlgs": ["RS256"],
         "usernameClaim": "sub",
         "usernamePrefix": "-",
         "requiredClaims": []
        },
        {
         "clientID": "updated-client-id",
         "issuerURL": "https://updated.issuer.com",
         "groupsClaim": "groups",
         "groupsPrefix": "-",
         "usernameClaim": "sub",
         "usernamePrefix": "acme-",
         "signingAlgs": ["RS256"]
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
     "clientID": "new-client-id",
     "issuerURL": "https://new.issuer.com",
     "groupsClaim": "groups",
     "groupsPrefix": "-",
     "usernameClaim": "sub",
     "usernamePrefix": "-",
     "signingAlgs": ["RS256"],
     "requiredClaims": []
    },
    {
     "clientID": "updated-client-id",
     "issuerURL": "https://updated.issuer.com",
     "groupsClaim": "groups",
     "groupsPrefix": "-",
     "usernameClaim": "sub",
     "usernamePrefix": "acme-",
     "signingAlgs": ["RS256"],
     "requiredClaims": []
    }
  ]
  ```

---

### Scenario 2: Updating an Instance with a Single OIDC Object

1. **Current OIDC Configuration**  
  The instance has the following OIDC object configuration:

  ```json
  {
    "clientID": "some-client-id",
    "issuerURL": "https://some.issuer.com",
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
      "issuerURL": "https://new.issuer.com",
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
    "issuerURL": "https://new.issuer.com",
    "groupsClaim": "groups",
    "groupsPrefix": "-",
    "usernameClaim": "sub",
    "usernamePrefix": "-",
    "signingAlgs": ["RS256"],
    "requiredClaims": []
  }
  ```

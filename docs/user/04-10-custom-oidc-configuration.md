# Custom OIDC Configuration

Configure a custom Open ID Connect (OIDC) as a list of `oidc` objects. Alternatively, for backward compatibility, you can configure it as a single `oidc` object.

To create SAP BTP, Kyma runtime with a custom OIDC configuration, you can specify either a list of `oidc` objects or a single `oidc` object as a provisioning parameter. While both options are supported, using a list of `oidc` objects is the recommended approach, even if you are defining only one OIDC configuration. The list allows you to define multiple OIDC configurations. The single `oidc` object is only supported for backward compatibility.

If you do not include an `oidc` list or a single `oidc` object in the provisioning request, the default OIDC configuration is applied. However, if you provide an empty `oidc` list with zero elements, no OIDC configuration is applied to the instance. 

The single `oidc` object defaults to the predefined values when its properties are left empty. However, you must explicitly define the `oidc` list because it does not inherit the default values for its items.

> [!NOTE]
> When using the `oidc` list for custom OIDC configuration, you must provide values for each element in the list except for **requiredClaims**. Otherwise, you get a provisioning error.

See an example with the OIDC list:

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
<summary>Configuration with a Single `oidc` Object</summary>

This solution is not recommended. It is only supported for backward compatibility with existing automations.

> [!NOTE]
> You must provide the **clientID** and **issuerURL**  values when using a single `oidc` object for custom OIDC configuration. Otherwise, you get a provisioning error.

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

### Configuration with No `oidc` Object and No List

To have the default OIDC configuration automatically applied, send a request without specifying any `oidc` configuration:

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

Result: You see a configuration with empty properties in the `oidc` object, similar to this one:

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

### Configuration with the `oidc` List

This method allows you to define one or multiple OIDC configurations, depending on the number of lists of `oidc` objects you add to your provisioning request.

To define one OIDC configuration, send a request with an `oidc` list containing a single configuration:

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

Result: You see a configuration similar to this one:

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

### Configuration with an Empty `oidc` Object

To apply the default OIDC configuration, send a request with an `oidc` object where all properties are left empty:

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

Result: You see the default OIDC configuration, similar to this one:

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

To update the OIDC configuration, provide values for the mandatory properties. Without these values, a validation error occurs. If you provide no `oidc` list and no single `oidc` object in the update request, the existing OIDC configuration remains unchanged. Providing an empty `oidc` list clears the OIDC configuration for the instance.

The update operation overwrites the OIDC configuration values provided in JSON for the `oidc` list, meaning that OIDC properties with empty values are considered valid and replace the existing values. However, for a single `oidc` object, empty values do not change the configuration, and only the provided values are updated. 

You can update the OIDC configuration from a single `oidc` object to an `oidc` list. However, updating from an `oidc` list to a single `oidc` object is not supported.


### Updating an Instance with an OIDC List of Objects

  The current OIDC object list configuration you want to update is the following :

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
  To update this OIDC configuration, send an HTTP PATCH request with the following payload:

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

Result: You see a configuration with the values provided in the request, similar to this one:

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

### Updating an Instance with a Single OIDC Object

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

1. To update the OIDC configuration, send an HTTP PATCH request with the following payload:

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

   Result: You see a configuration with the values provided in the request, similar to this one:

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

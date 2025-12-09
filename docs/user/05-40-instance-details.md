# Check SAP BTP, Kyma Runtime Instance Details

Get the SAP BTP, Kyma runtime instance details.

## Steps

1. Export the instance ID that you set during [provisioning](05-10-provisioning-kyma-environment.md):

   ```bash
   export INSTANCE_ID={SET_INSTANCE_ID}
   ```

   > ### Note:
   > Ensure that the BROKER_URL and INSTANCE_ID environment variables are exported as well before you proceed.

2. Make a call to Kyma Environment Broker with a proper **Authorization** [request header](../contributor/01-10-authorization.md) to verify that provisioning/deprovisioning succeeded:

   ```bash
   curl --request GET "https://$BROKER_URL/oauth/v2/service_instances/$INSTANCE_ID" \
   --header 'X-Broker-API-Version: 2.14' \
   --header "$AUTHORIZATION_HEADER"
   ```

   A successful call returns the instance details:

      ```json
      {
         "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
         "plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
         "dashboard_url": "https://console.{DOMAIN}",
         "parameters": {
            "plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
            "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
            "ers_context": {
               "tenant_id": "5b6a5174-001b-41f6-8884-9bfece8a2dae",
               "subaccount_id": "",
               "globalaccount_id": "",
               "active": true,
               "user_id": "user@sap.com",
               "commercial_model": "commercial model",
               "license_type": "license",
               "origin": "origin",
               "platform": "platform",
               "region": "us10-staging"
            }
         },
         "parameters": {
            "autoScalerMax": 3,
            "autoScalerMin": 1,
            "oidc": {
               "clientID": "",
               "groupsClaim": "",
               "issuerURL": "",
               "signingAlgs": [],
               "usernameClaim": "",
               "usernamePrefix": ""
            },
            "networking": {
               "nodes": "10.250.0.0/22",
               "pods": "10.96.0.0/13",
               "services": "10.104.0.0/13"
            }
         },
         "platform_region": "cf-us10-staging",
         "platform_provider": "AWS"
      }
      ```

  > ### Note:
  > The fields under the **parameters** field can differ depending on the provisioning input.

# Restricted Global Accounts
## Overview

The Kyma Environment Broker configuration allows to restrict the provisioning for a defined set of Global Accounts.
Each provisioning request contains the Global Account ID as a part of the context (the **globalaccount_id** field), for example:

```json
{
      "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
      "plan_id": "4deee563-e5ec-4731-b9b1-53b42d855f0c",
      "context": {
         "globalaccount_id": "2f5011af-2fd3-44ba-ac60-eeb1148c2995",
         "subaccount_id": "8b9a0db4-9aef-4da2-a856-61a4420b66fd",
      },
      "parameters": {
         "name": "azure-cluster",
         "region": "northeurope"
      }
   }
```

## Configuration
To enable the restriction, set the following properties:
- **allowedGlobalAccountIDs** - a comma-separated list of allowed Global Account IDs.
- **restrictToAllowedGlobalAccountIDs** - a boolean flag to enable or disable the restriction.

See an example of the restriction configuration:

```yaml
broker:
  allowedGlobalAccountIDs: "2f5011af-2fd3-44ba-ac60-eeb1148c2995,another-global-account-id"
  restrictToAllowedGlobalAccountIDs: true
```

When the restriction is enabled, KEB checks if the **globalaccount_id** from the provisioning request context is contained in the list of allowed Global Account IDs. If not, the provisioning request is rejected with an appropriate error message.


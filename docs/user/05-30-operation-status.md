# Check Operation Status

Check the operation status for the provisioning and deprovisioning operations.

## Steps

1. Export the operation ID that you obtained during [provisioning](05-10-provisioning-kyma-environment.md) or [deprovisioning](05-20-deprovisioning-kyma-environment.md) as an environment variable:

   ```bash
   export OPERATION_ID={OBTAINED_OPERATION_ID}
   ```

   > ### Note:
   > Ensure that the **BROKER_URL** and **INSTANCE_ID** environment variables are exported as well before you proceed.

2. Make a call to Kyma Environment Broker with a proper **Authorization** [request header](../contributor/01-10-authorization.md) to verify that provisioning or deprovisioning succeeded.

   ```bash
   curl --request GET "https://$BROKER_URL/oauth/v2/service_instances/$INSTANCE_ID/last_operation?operation=$OPERATION_ID&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281" \
   --header 'X-Broker-API-Version: 2.13' \
   --header "$AUTHORIZATION_HEADER"
   ```

   A successful call returns the operation status and description:

      ```json
      {
         "state": "succeeded",
         "description": "Operation succeeded."
      }
      ```

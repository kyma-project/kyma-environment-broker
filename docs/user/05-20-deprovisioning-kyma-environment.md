# Deprovision SAP BTP, Kyma Runtime Using Kyma Environment Broker

Deprovision SAP BTP, Kyma runtime on Microsoft Azure using Kyma Environment Broker (KEB).

## Procedure

1. Ensure that these environment variables are exported:

   ```bash
   export BROKER_URL={KYMA_ENVIRONMENT_BROKER_URL}
   export INSTANCE_ID={INSTANCE_ID_FROM_PROVISIONING_CALL}
   ```

2. Get the [access token](../contributor/01-10-authorization.md#get-the-access-token). Export this variable based on the token you got from the OAuth client.

   ```bash
   export AUTHORIZATION_HEADER="Authorization: Bearer $ACCESS_TOKEN"
   ```

3. Make a call to KEB to delete a Kyma runtime instance on Microsoft Azure.

   ```bash
   curl  --request DELETE "https://$BROKER_URL/oauth/v2/service_instances/$INSTANCE_ID?accepts_incomplete=true&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281&plan_id=4deee563-e5ec-4731-b9b1-53b42d855f0c" \
   --header 'X-Broker-API-Version: 2.13' \
   --header "$AUTHORIZATION_HEADER"
   ```

A successful call returns the operation ID.

   ```json
   {
       "operation":"8a7bfd9b-f2f5-43d1-bb67-177d2434053c"
   }
   ```

4. Check the operation status as described in the [Check Operation Status](05-30-operation-status.md) document.

## Subaccount Cleanup Job

The standard workflow for the [SAP BTP service operator](https://github.com/SAP/sap-btp-service-operator) resources is to keep them untouched by KEB because users may intend to
keep the external services provisioned through the SAP BTP service operator still operational. In this case, when calling deprovisioning in the SAP BTP cockpit, users are informed
there are still instances provisioned by the SAP BTP service operator, and the user is expected to handle the cleanup.

There is one exception, and that is the [Subaccount Cleanup CronJob](../contributor/06-30-subaccount-cleanup-cronjob.md). KEB [parses the `User-Agent` HTTP header](https://github.com/kyma-project/kyma-environment-broker/blob/main/internal/process/deprovisioning/btp_operator_cleanup.go#L87) for a
`DELETE` call on the `/service_instances/${instance_id}` endpoint and forwards it through the operation to the processing step `btp_operator_cleanup` handling
soft delete for existing SAP BTP service operator resources. Since the `subaccount-cleanup` Job is triggered automatically and deletes only Kyma runtimes where the whole subaccount is
intended for deletion, it is necessary to execute the SAP BTP service operator cleanup procedure as well.

<!--{"metadata":{"publish":true}}-->

# Subaccount Cleanup CronJob

Each SAP BTP, Kyma runtime instance in the Kyma Environment Broker (KEB) database belongs to a global account and to a subaccount.
Subaccount Cleanup is an application that periodically calls the CIS service and reports **Subaccount_Deletion** events.
Based on these events, Subaccount Cleanup triggers the deprovisioning action on Kyma runtime instances belonging to the given subaccount.

## Details

The Subaccount Cleanup workflow is divided into several steps:

1. Fetch **Subaccount_Deletion** events from the CIS service.

   The behavior depends on the configured Events Service version (**APP_EVENTS_SERVICE_VERSION**):

   - **CIS v1** (legacy):
     1. The CIS client calls the CIS service and receives a paginated list of events.
     2. It fetches remaining pages one by one, identified by page number.
     3. A subaccount ID is extracted from each event and collected into an array.
     4. Once complete, the client logs the number of subaccounts fetched and the time range of events.

   - **CIS v2** (default):
     1. The CIS client calls `/events/v2/events/central` requesting **Subaccount_Deletion** events for the `Subaccount` entity type, covering the last 30 days.
     2. It fetches subsequent pages by following the **nextCursor** value in each response, until no cursor is returned.
     3. A subaccount ID is extracted from each event and collected into an array.
     4. Once complete, the client logs the number of subaccounts fetched and the time range of events.

2. Find all instances in the KEB database based on the fetched subaccount IDs.
   The subaccounts pool is divided into batches. For each batch, a query is made to the database to fetch instances.

3. Trigger the deprovisioning operation for each instance found in step 2.

   Logs provide the status of each triggered action:

    ```
    deprovisioning for instance <InstanceID> (SubAccountID: <SubAccountID>) was triggered, operation: <OperationID>
    ```

   Subaccount Cleanup also uses logs to inform about the completion of the deprovisioning operation.

## Prerequisites

* CIS service to receive all **Subaccount_Deletion** events
* The KEB database to get the instance ID for each subaccount ID from the **Subaccount_Deletion** event
* KEB to trigger Kyma runtime instance deprovisioning

## Configuration

Use the following environment variables to configure the application:

| Environment Variable | Current Value | Description |
|---------------------|------------------------------|---------------------------------------------------------------|
| **APP_BROKER_URL** | None | - |
| **APP_CIS_AUTH_URL** | <code>TBD</code> | The OAuth2 token endpoint (authorization URL) for CIS v2, used to obtain access tokens for authenticating requests. |
| **APP_CIS_CLIENT_ID** | None | Specifies the client ID for the OAuth2 authentication in CIS. |
| **APP_CIS_CLIENT_&#x200b;SECRET** | None | Specifies the client secret for the OAuth2 authentication in CIS. |
| **APP_CIS_EVENT_&#x200b;SERVICE_URL** | <code>TBD</code> | The endpoint URL for the CIS v2 event service, used to fetch subaccount events. |
| **APP_CIS_MAX_REQUEST_&#x200b;RETRIES** | <code>3</code> | The maximum number of request retries to the CIS v2 API in case of errors. |
| **APP_CIS_RATE_&#x200b;LIMITING_INTERVAL** | <code>2s</code> | The minimum interval between requests to the CIS v2 API in case of errors. |
| **APP_CIS_REQUEST_&#x200b;INTERVAL** | <code>200ms</code> | The interval between requests to the CIS v2 API. |
| **APP_EVENTS_SERVICE_&#x200b;VERSION** | <code>v2</code> | Specifies the Events Service version. |
| **APP_DATABASE_HOST** | None | Specifies the host of the database. |
| **APP_DATABASE_NAME** | None | Specifies the name of the database. |
| **APP_DATABASE_&#x200b;PASSWORD** | None | Specifies the user password for the database. |
| **APP_DATABASE_PORT** | None | Specifies the port for the database. |
| **APP_DATABASE_SECRET_&#x200b;KEY** | None | Specifies the Secret key for the database. |
| **APP_DATABASE_SSLMODE** | None | Activates the SSL mode for PostgreSQL. |
| **APP_DATABASE_&#x200b;SSLROOTCERT** | <code>/secrets/cloudsql-sslrootcert/server-ca.pem</code> | Path to the Cloud SQL SSL root certificate file. |
| **APP_DATABASE_USER** | None | Specifies the username for the database. |
| **DATABASE_EMBEDDED** | <code>true</code> | - |

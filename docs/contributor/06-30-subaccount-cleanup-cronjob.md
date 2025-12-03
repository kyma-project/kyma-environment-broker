# Subaccount Cleanup CronJob

Each SAP BTP, Kyma runtime instance in the Kyma Environment Broker (KEB) database belongs to a global account and to a subaccount.
Subaccount Cleanup is an application that periodically calls the CIS service and notifies about `SUBACCOUNT_DELETE` events.
Based on these events, Subaccount Cleanup triggers the deprovisioning action on the Kyma runtime instances belonging to the given subaccount.

## Details

The Subaccount Cleanup workflow is divided into several steps:

1. Fetch `SUBACCOUNT_DELETE` events from the CIS service.

    a. CIS client makes a call to the CIS service and as a response, it gets a list of events divided into pages.

    b. CIS client fetches the rest of the events by making a call to each page one by one.

    c. A subaccount ID is taken from each event and kept in an array.

    d. When the CIS client ends its workflow, it displays logs with information on how many subaccounts were fetched.

2. Find all instances in the KEB database based on the fetched subaccount IDs.
   The subaccounts pool is divided into pieces. For each piece, a query is made to the database to fetch instances.

3. Trigger the deprovisioning operation for each instance found in step 2.

   Logs inform about the status of each triggered action:

    ```
    deprovisioning for instance <InstanceID> (SubAccountID: <SubAccountID>) was triggered, operation: <OperationID>
    ```

   Subaccount Cleanup also uses logs to inform about the end of the deprovisioning operation.

## Prerequisites

* CIS service to receive all **SUBACCOUNT_DELETE** events
* The KEB database to get the instance ID for each subaccount ID from the **SUBACCOUNT_DELETE** event
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
| **APP_CLIENT_VERSION** | <code>v2.0</code> | Client version. |
| **APP_DATABASE_FIPS_&#x200b;WRITE_GCM** | <code>false</code> | Write using AES-GCM encryption. |
| **APP_DATABASE_HOST** | None | Specifies the host of the database. |
| **APP_DATABASE_NAME** | None | Specifies the name of the database. |
| **APP_DATABASE_&#x200b;PASSWORD** | None | Specifies the user password for the database. |
| **APP_DATABASE_PORT** | None | Specifies the port for the database. |
| **APP_DATABASE_SECRET_&#x200b;KEY** | None | Specifies the Secret key for the database. |
| **APP_DATABASE_SSLMODE** | None | Activates the SSL mode for PostgreSQL. |
| **APP_DATABASE_&#x200b;SSLROOTCERT** | <code>/secrets/cloudsql-sslrootcert/server-ca.pem</code> | Path to the Cloud SQL SSL root certificate file. |
| **APP_DATABASE_USER** | None | Specifies the username for the database. |
| **DATABASE_EMBEDDED** | <code>true</code> | - |

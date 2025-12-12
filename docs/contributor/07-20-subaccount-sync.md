# Subaccount Sync

Subaccount Sync is an application that performs reconciliation tasks on SAP BTP, Kyma runtime, synchronizing Kyma custom
resource (CR) labels with subaccount attributes.

## Details

The `operator.kyma-project.io/beta` and `operator.kyma-project.io/used-for-production` labels of all Kyma CRs for a given subaccount are synchronized 
with the `Enable beta features` and `Used for Production` attributes accordingly.
The current state of the attributes is persisted in the `subaccount_states` database table.

The table structure:

| Column name             | Type         | Description                                               |
|-------------------------|--------------|-----------------------------------------------------------|
| **id**                  | VARCHAR(255) | Subaccount ID                                             |
| **enable_beta**         | VARCHAR(255) | Enable beta                                               |
| **used_for_production** | VARCHAR(255) | Used for production                                       |
| **modified_at**         | BIGINT       | Last modification timestamp as Unix epoch in milliseconds |

The application periodically:

* Fetches data for selected subaccounts from CIS Account service
* Fetches events from CIS Event service for configurable time window
* Monitors Kyma CRs using informer and detects changes in the labels
* Persists the desired (set in CIS) state of the attributes in the database
* Updates the labels of the Kyma CRs if the state of the attributes has changed

## Prerequisites

* The KEB Go packages so that Subaccount Sync can reuse them
* The KEB database for storing current state of selected attributes
### Dry Run Mode

The dry run mode does not perform any changes on the control plane. Setting **SUBACCOUNT_SYNC_UPDATE_RESOURCES** to `false` runs the application in dry run mode.
Updater is not created and no changes are made to the Kyma CRs. The application only fetches
data from CIS and updates the database.
Differences between the desired and current state of the attributes cause that the queue is filled with entries.
Since this is an augmented queue with one entry for each subaccount, the length does not exceed the number of subaccounts.

### Resources

* Subaccount-sync deployment defined in [subaccount-sync-deployment.yaml](../../resources/keb/templates/subaccount-sync-deployment.yaml) - deployment configuration
* Subaccount-sync service defined in [service.yaml](../../resources/keb/templates/service.yaml) - service configuration, required for metrics scraping
* Subaccount-sync VMServiceScrape defined in [service-monitor.yaml](../../resources/keb/templates/service-monitor.yaml) - Prometheus scrape configuration referring to the service required for metrics scraping
* Subaccount-sync PeerAuthentication defined in [policy.yaml](../../resources/keb/templates/policy.yaml) - PeerAuthentication configuration required for metrics scraping

## Configuration

The application is defined as a Kubernetes deployment.

Use the following environment variables to configure the application:

| Environment Variable | Current Value | Description |
|---------------------|------------------------------|---------------------------------------------------------------|
| **SUBACCOUNT_SYNC_&#x200b;ACCOUNTS_SYNC_&#x200b;INTERVAL** | <code>24h</code> | Interval between full account synchronization runs. |
| **SUBACCOUNT_SYNC_&#x200b;ALWAYS_SUBACCOUNT_&#x200b;FROM_DATABASE** | <code>false</code> | If true, fetches subaccountID from the database only when the subaccount is empty. |
| **SUBACCOUNT_SYNC_CIS_&#x200b;ACCOUNTS_AUTH_URL** | <code>TBD</code> | The OAuth2 token endpoint (authorization URL) used to obtain access tokens for authenticating requests to the CIS Accounts API. |
| **SUBACCOUNT_SYNC_CIS_&#x200b;ACCOUNTS_CLIENT_ID** | None | Specifies the **CLIENT_ID** for the client accessing accounts. |
| **SUBACCOUNT_SYNC_CIS_&#x200b;ACCOUNTS_CLIENT_&#x200b;SECRET** | None | Specifies the **CLIENT_SECRET** for the client accessing accounts. |
| **SUBACCOUNT_SYNC_CIS_&#x200b;ACCOUNTS_MAX_&#x200b;REQUESTS_PER_&#x200b;INTERVAL** | <code>5</code> | Maximum number of requests per interval to the CIS Accounts API. |
| **SUBACCOUNT_SYNC_CIS_&#x200b;ACCOUNTS_RATE_&#x200b;LIMITING_INTERVAL** | <code>2s</code> | Minimum interval between requests to the CIS Accounts API. |
| **SUBACCOUNT_SYNC_CIS_&#x200b;ACCOUNTS_SERVICE_URL** | <code>TBD</code> | The base URL of the CIS Accounts API endpoint, used for fetching subaccount data. |
| **SUBACCOUNT_SYNC_CIS_&#x200b;EVENTS_AUTH_URL** | <code>TBD</code> | The OAuth2 token endpoint (authorization URL) for CIS v2, used to obtain access tokens for authenticating requests. |
| **SUBACCOUNT_SYNC_CIS_&#x200b;EVENTS_CLIENT_ID** | None | Specifies the **CLIENT_ID** for client accessing events. |
| **SUBACCOUNT_SYNC_CIS_&#x200b;EVENTS_CLIENT_SECRET** | None | Specifies the **CLIENT_SECRET** for the client accessing events. |
| **SUBACCOUNT_SYNC_CIS_&#x200b;EVENTS_MAX_REQUESTS_&#x200b;PER_INTERVAL** | <code>5</code> | Maximum number of requests per interval to the CIS Events API. |
| **SUBACCOUNT_SYNC_CIS_&#x200b;EVENTS_RATE_&#x200b;LIMITING_INTERVAL** | <code>2s</code> | Minimum interval between requests to the CIS Events API. |
| **SUBACCOUNT_SYNC_CIS_&#x200b;EVENTS_SERVICE_URL** | <code>TBD</code> | The endpoint URL for the CIS v2 event service, used to fetch subaccount events. |
| **SUBACCOUNT_SYNC_&#x200b;DATABASE_FIPS_WRITE_&#x200b;GCM** | <code>true</code> | Write using AES-GCM encryption. |
| **SUBACCOUNT_SYNC_&#x200b;DATABASE_HOST** | None | Specifies the host of the database. |
| **SUBACCOUNT_SYNC_&#x200b;DATABASE_NAME** | None | Specifies the name of the database. |
| **SUBACCOUNT_SYNC_&#x200b;DATABASE_PASSWORD** | None | Specifies the user password for the database. |
| **SUBACCOUNT_SYNC_&#x200b;DATABASE_PORT** | None | Specifies the port for the database. |
| **SUBACCOUNT_SYNC_&#x200b;DATABASE_SECRET_KEY** | None | Specifies the Secret key for the database. |
| **SUBACCOUNT_SYNC_&#x200b;DATABASE_SSLMODE** | None | Activates the SSL mode for PostgreSQL. |
| **SUBACCOUNT_SYNC_&#x200b;DATABASE_SSLROOTCERT** | <code>/secrets/cloudsql-sslrootcert/server-ca.pem</code> | Path to the Cloud SQL SSL root certificate file. |
| **SUBACCOUNT_SYNC_&#x200b;DATABASE_USER** | None | Specifies the username for the database. |
| **SUBACCOUNT_SYNC_&#x200b;EVENTS_WINDOW_&#x200b;INTERVAL** | <code>15m</code> | Time window for collecting events from CIS. |
| **SUBACCOUNT_SYNC_&#x200b;EVENTS_WINDOW_SIZE** | <code>20m</code> | Size of the time window for collecting events from CIS. |
| **SUBACCOUNT_SYNC_LOG_&#x200b;LEVEL** | <code>info</code> | Log level for the subaccount sync job. |
| **SUBACCOUNT_SYNC_&#x200b;METRICS_PORT** | <code>8081</code> | Port on which the subaccount sync service exposes Prometheus metrics. |
| **SUBACCOUNT_SYNC_&#x200b;QUEUE_SLEEP_INTERVAL** | <code>30s</code> | Interval between queue processing cycles. |
| **SUBACCOUNT_SYNC_&#x200b;RUNTIME_&#x200b;CONFIGURATION_&#x200b;CONFIG_MAP_NAME** | None | Name of the ConfigMap with the default KymaCR template. |
| **SUBACCOUNT_SYNC_&#x200b;STORAGE_SYNC_&#x200b;INTERVAL** | <code>5m</code> | Interval between storage synchronization. |
| **SUBACCOUNT_SYNC_&#x200b;UPDATE_RESOURCES** | <code>false</code> | If true, enables updating resources during subaccount sync. |

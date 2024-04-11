# Subaccount Sync

Subaccount Sync is an application that performs reconciliation tasks on SAP BTP, Kyma runtime, synchronizing Kyma custom
resource (CR) labels with subaccount attributes.

## Details

The `operator.kyma-project.io/beta` label of all Kyma CRs for a given subaccount is synchronized with
the `Enable beta features` attribute of this subaccount.
Current state of the attribute is persisted in the database table `subaccount_states`.
The `Used for production` is monitored as well and the state is persisted in the same table, however it does not affect
any resources.

The table structure:

| Column name             | Type         | Description                                               |
|-------------------------|--------------|-----------------------------------------------------------|
| **id**                  | VARCHAR(255) | Subaccount ID                                             |
| **enable_beta**         | VARCHAR(255) | Enable beta                                               |
| **used_for_production** | VARCHAR(255) | Used for production                                       |
| **modified_at**         | BIGINT       | Last modification timestamp as Unix epoch in milliseconds |


The application periodically:

- Fetches data for selected subaccounts from CIS Account service
- Fetches events from CIS Event service for configurable time window
- Monitors Kyma CRs using informer and detects changes in the labels
- Persists the desired (set in CIS) state of the attributes in the database
- Updates the labels of the Kyma CRs if the state of the attributes has changed

## Prerequisites

- The KEB Go packages so that Subaccount Sync can reuse them
- The KEB database for storing current state of selected attributes

## Configuration

The application is defined as a Kubernetes deployment.

Use the following environment variables to configure the application:

| Environment variable                                       | Description                                                                                                               | Default value |
|------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------|---------------|
| **SUBACCOUNT_SYNC_KYMA_VERSION**                           | Specifies current Kyma version to get proper Kyma Custom Resource Definition.                                             | `false`       |
| **SUBACCOUNT_SYNC_METRICS_PORT**                           | Specifies port where metrics are exposed for scrapion.                                                                    | `8081`        |
| **SUBACCOUNT_SYNC_UPDATE_RESOURCES**                       | Specifies whether to run the updater process which updates Kyma Custom Resources.                                         | `false`       |
| **SUBACCOUNT_SYNC_LOG_LEVEL**                              | Specifies log level.                                                                                                      | `info`        |
| **SUBACCOUNT_SYNC_ACCOUNTS_SYNC_INTERVAL**                 | Specifies at what intervals subaccounts data is fetched.                                                                  | `24h`         |
| **SUBACCOUNT_SYNC_STORAGE_SYNC_INTERVAL**                  | Specifies at what intervals subaccount states are persisted in database.                                                  | `2m`          |
| **SUBACCOUNT_SYNC_EVENTS_WINDOW_SIZE**                     | Specifies size of events window.                                                                                          | `20m`         |
| **SUBACCOUNT_SYNC_EVENTS_WINDOW_INTERVAL**                 | Specifies at what intervals we fetch events.                                                                              | `15m`         |
| **SUBACCOUNT_SYNC_QUEUE_SLEEP_INTERVAL**                   | Specifies how long the updater sleeps if queue is empty.                                                                  | `30s`         |
| **SUBACCOUNT_SYNC_CIS_EVENTS_CLIENT_ID**                   | Specifies the CLIENT_ID for client accessing events.                                                                      |               |
| **SUBACCOUNT_SYNC_CIS_EVENTS_CLIENT_SECRET**               | Specifies the CLIENT_SECRET for client accessing events.                                                                  |               |
| **SUBACCOUNT_SYNC_CIS_EVENTS_AUTH_URL**                    | Specifies the authorization URL for events endpoint.                                                                      |               |
| **SUBACCOUNT_SYNC_CIS_EVENTS_SERVICE_URL**                 | Specifies the URL for events endpoint.                                                                                    |               |
| **SUBACCOUNT_SYNC_CIS_EVENTS_RATE_LIMITING_INTERVAL**      | Specifies the rate limiting interval for events endpoint.                                                                 | `2s`          |
| **SUBACCOUNT_SYNC_CIS_EVENTS_MAX_REQUESTS_PER_INTERVAL**   | Specifies the number of allowed requests per interval for events endpoint.                                                | 5             |
| **SUBACCOUNT_SYNC_CIS_ACCOUNTS_CLIENT_ID**                 | Specifies the CLIENT_ID for client accessing accounts.                                                                    |               |
| **SUBACCOUNT_SYNC_CIS_ACCOUNTS_CLIENT_SECRET**             | Specifies the CLIENT_SECRET for client accessing accounts.                                                                |               |
| **SUBACCOUNT_SYNC_CIS_ACCOUNTS_AUTH_URL**                  | Specifies the authorization URL for accounts endpoint.                                                                    |               |
| **SUBACCOUNT_SYNC_CIS_ACCOUNTS_SERVICE_URL**               | Specifies the URL for accounts endpoint.                                                                                  |               |
| **SUBACCOUNT_SYNC_CIS_ACCOUNTS_RATE_LIMITING_INTERVAL**    | Specifies the rate limiting interval for accounts endpoint.                                                               | `2s`          |
| **SUBACCOUNT_SYNC_CIS_ACCOUNTS_MAX_REQUESTS_PER_INTERVAL** | Specifies the number of allowed requests per interval for accounts endpoint.                                              | 5             |
| **SUBACCOUNT_SYNC_DATABASE_SECRET_KEY**                    | Specifies the secret key for the database.                                                                                | optional      |
| **SUBACCOUNT_SYNC_DATABASE_USER**                          | Specifies the username for the database.                                                                                  | `postgres`    |
| **SUBACCOUNT_SYNC_DATABASE_PASSWORD**                      | Specifies the user password for the database.                                                                             | `password`    |
| **SUBACCOUNT_SYNC_DATABASE_HOST**                          | Specifies the host of the database.                                                                                       | `localhost`   |
| **SUBACCOUNT_SYNC_DATABASE_PORT**                          | Specifies the port for the database.                                                                                      | `5432`        |
| **SUBACCOUNT_SYNC_DATABASE_NAME**                          | Specifies the name of the database.                                                                                       | `broker`      |
| **SUBACCOUNT_SYNC_DATABASE_SSLMODE**                       | Activates the SSL mode for PostgreSQL. See [all the possible values](https://www.postgresql.org/docs/9.1/libpq-ssl.html). | `disable`     |
| **SUBACCOUNT_SYNC_DATABASE_SSLROOTCERT**                   | Specifies the location of CA cert of PostgreSQL. (Optional)                                                               | optional      |

### Dry Run mode

The dry run mode does not perform any changes on the control plane. Setting `SUBACCOUNT_SYNC_UPDATE_RESOURCES` to false
will run the application in dry run mode.
Updater would not be created and no changes would be made to the Kyma Custom Resources. The application will only fetch
data from CIS and update the database.
Differences between the desired and current state of the attributes will cause queue to be filled with entries.
Since this is augmented queue with one entry for each subaccount, the length will not exceed the number of subaccounts.


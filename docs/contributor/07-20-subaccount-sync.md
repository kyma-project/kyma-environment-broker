# Subaccount Sync

Subaccount Sync is an application that performs reconciliation tasks on SAP BTP, Kyma runtime, synchronizing Kyma Custom Resource labels with subaccount attributes.

## Details

The `operator.kyma-project.io/beta` labels of all Kyma CRs for given subaccount are synchronized with the `Enable beta features` attribute of this subaccount. 
Current state of the attribute is persisted in the database table `subaccount_states`.
The `Used for production` is monitored as well and the state is persisted in the same table, however it does not affect any resources.

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

| Environment variable                                             | Description                                                                                                                      | Default value |
| ---------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------- | ------------- |
| **SUBACCOUNT_SYNC_WATCHER_ENABLED**                           | Specifies whether the application should use Runtime Watcher for reconciliation.                                                                   | `false`        |
| **SUBACCOUNT_SYNC_JOB_ENABLED**                               | Specifies whether the application should use the job to reconcile.                                                                       | `false`        |
| **SUBACCOUNT_SYNC_DRY_RUN**                                   | Specifies whether to run the application in the dry-run mode.                                                                    | `true`        |
| **SUBACCOUNT_SYNC_BTP_MANAGER_SECRET_WATCHER_ADDR**           | Specifies Runtime Watcher's port.                                                                                                       | `0`           |
| **SUBACCOUNT_SYNC_BTP_MANAGER_SECRET_WATCHER_COMPONENT_NAME** | Specifies the component name for Runtime Watcher.                                                                                               | `NA`          |
| **SUBACCOUNT_SYNC_AUTO_RECONCILE_INTERVAL**                   | Specifies at what intervals the job runs  (in hours).                                                                       | `24`          |
| **SUBACCOUNT_SYNC_DATABASE_SECRET_KEY**                       | Specifies the secret key for the database.                                                                                       | optional      |
| **SUBACCOUNT_SYNC_DATABASE_USER**                             | Specifies the username for the database.                                                                                         | `postgres`    |
| **SUBACCOUNT_SYNC_DATABASE_PASSWORD**                         | Specifies the user password for the database.                                                                                    | `password`    |
| **SUBACCOUNT_SYNC_DATABASE_HOST**                             | Specifies the host of the database.                                                                                              | `localhost`   |
| **SUBACCOUNT_SYNC_DATABASE_PORT**                             | Specifies the port for the database.                                                                                             | `5432`        |
| **SUBACCOUNT_SYNC_DATABASE_NAME**                             | Specifies the name of the database.                                                                                              | `broker`      |
| **SUBACCOUNT_SYNC_DATABASE_SSLMODE**                          | Activates the SSL mode for PostgreSQL. See [all the possible values](https://www.postgresql.org/docs/9.1/libpq-ssl.html).       | `disable`     |
| **SUBACCOUNT_SYNC_DATABASE_SSLROOTCERT**                      | Specifies the location of CA cert of PostgreSQL. (Optional)                                                                      |  optional     |

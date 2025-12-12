# Runtime Reconciler

Runtime Reconciler is an application that performs reconciliation tasks on SAP BTP, Kyma runtime.

## Details

Runtime Reconciler reconciles BTP Manager Secrets on Kyma runtimes with a job, 
which periodically loops over all instances from the KEB database. Each instance has an existing assigned Runtime ID. 
The job checks if the Secret on the Kyma runtime matches the credentials from the KEB database.

> ### Note:
> If you modify or delete the `sap-btp-manager` Secret, it is modified back to its previous settings or regenerated within up to 24 hours. However, if the Secret is labeled with `kyma-project.io/skip-reconciliation: "true"`, the job skips the reconciliation for this Secret.
> To revert the Secret to its default state (stored in the KEB database), restart Runtime Reconciler, for example, by scaling down the deployment to `0` and then back to `1`.

## Prerequisites

* The KEB Go packages so that Runtime Reconciler can reuse them
* The KEB database for storing the Kubernetes Secrets that match the Secrets on Kyma runtimes

## Configuration

The application is defined as a Kubernetes deployment.

Use the following environment variables to configure the application:

| Environment Variable | Current Value | Description |
|---------------------|------------------------------|---------------------------------------------------------------|
| **RUNTIME_RECONCILER_&#x200b;DATABASE_FIPS_WRITE_&#x200b;GCM** | <code>true</code> | Write using AES-GCM encryption. |
| **RUNTIME_RECONCILER_&#x200b;DATABASE_HOST** | None | Specifies the host of the database. |
| **RUNTIME_RECONCILER_&#x200b;DATABASE_NAME** | None | Specifies the name of the database. |
| **RUNTIME_RECONCILER_&#x200b;DATABASE_PASSWORD** | None | Specifies the user password for the database. |
| **RUNTIME_RECONCILER_&#x200b;DATABASE_PORT** | None | Specifies the port for the database. |
| **RUNTIME_RECONCILER_&#x200b;DATABASE_SECRET_KEY** | None | Specifies the Secret key for the database. |
| **RUNTIME_RECONCILER_&#x200b;DATABASE_SSLMODE** | None | Activates the SSL mode for PostgreSQL. |
| **RUNTIME_RECONCILER_&#x200b;DATABASE_SSLROOTCERT** | <code>/secrets/cloudsql-sslrootcert/server-ca.pem</code> | Path to the Cloud SQL SSL root certificate file. |
| **RUNTIME_RECONCILER_&#x200b;DATABASE_USER** | None | Specifies the username for the database. |
| **RUNTIME_RECONCILER_&#x200b;DRY_RUN** | <code>false</code> | If true, runs the reconciler in dry-run mode (no changes are made, only logs actions). |
| **RUNTIME_RECONCILER_&#x200b;JOB_ENABLED** | <code>false</code> | If true, enables the periodic reconciliation job. |
| **RUNTIME_RECONCILER_&#x200b;JOB_INTERVAL** | <code>1440</code> | Interval (in minutes) between reconciliation job runs. |
| **RUNTIME_RECONCILER_&#x200b;JOB_RECONCILIATION_&#x200b;DELAY** | <code>1s</code> | Delay before starting reconciliation after job trigger. |
| **RUNTIME_RECONCILER_&#x200b;METRICS_PORT** | <code>8081</code> | Port on which the reconciler exposes Prometheus metrics. |

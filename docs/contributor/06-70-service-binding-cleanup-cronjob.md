# Service Binding Cleanup CronJob

Use Service Binding Cleanup CronJob to remove expired service bindings for SAP BTP, Kyma runtime instances.

## Details

For each expired service binding, a DELETE request is sent to Kyma Environment Broker (KEB). The request has a time limit and can be retried if it times out.

### Dry-Run Mode

If you need to test the Job, run it in dry-run mode.
In this mode, the Job only logs the information about the number of expired service bindings without sending DELETE requests to KEB.

## Prerequisites

* The KEB database to get the IDs of expired service bindings
* KEB to initiate the service binding deletion process

## Configuration

The Job is a CronJob with a schedule that can be configured as a value in the [values.yaml](https://github.com/kyma-project/kyma-environment-broker/blob/main/resources/keb/values.yaml) file for the chart (see [Schedule syntax](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/#schedule-syntax)).
By default, the CronJob is scheduled as follows:

```yaml  
kyma-environment-broker.serviceBindingCleanup.schedule: "0 2,14 * * *"
```

Use the following environment variables to configure the Job:

| Environment Variable | Current Value | Description |
|---------------------|------------------------------|---------------------------------------------------------------|
| **APP_BROKER_URL** | None | - |
| **APP_DATABASE_HOST** | None | Specifies the host of the database. |
| **APP_DATABASE_NAME** | None | Specifies the name of the database. |
| **APP_DATABASE_&#x200b;PASSWORD** | None | Specifies the user password for the database. |
| **APP_DATABASE_PORT** | None | Specifies the port for the database. |
| **APP_DATABASE_SECRET_&#x200b;KEY** | None | Specifies the Secret key for the database. |
| **APP_DATABASE_SSLMODE** | None | Activates the SSL mode for PostgreSQL. |
| **APP_DATABASE_&#x200b;SSLROOTCERT** | <code>/secrets/cloudsql-sslrootcert/server-ca.pem</code> | Path to the Cloud SQL SSL root certificate file. |
| **APP_DATABASE_USER** | None | Specifies the username for the database. |
| **APP_JOB_DRY_RUN** | <code>false</code> | If true, the Job only logs what would be deleted without actually removing any bindings. |
| **APP_JOB_REQUEST_&#x200b;RETRIES** | <code>2</code> | Number of times to retry a failed DELETE request for a binding. |
| **APP_JOB_REQUEST_&#x200b;TIMEOUT** | <code>2s</code> | Timeout for each DELETE request to the broker. |
| **DATABASE_EMBEDDED** | <code>true</code> | - |

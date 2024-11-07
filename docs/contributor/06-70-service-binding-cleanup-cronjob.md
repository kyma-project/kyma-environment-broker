# Service Binding Cleanup CronJob

Use Service Binding Cleanup CronJob to remove expired service bindings for SAP BTP, Kyma runtime instances.

## Details

For each expired service binding, a DELETE request is sent to Kyma Environment Broker (KEB). The request has a time limit and can be retried if it timeouts.

### Dry-Run Mode

If you need to test the Job, run it in the `dry-run` mode.
In that mode, the Job only logs the information about the number of expired service bindings without sending the DELETE requests to KEB.

## Prerequisites

The Job requires access to:

* the KEB database to get the IDs of the expired service bindings
* KEB to initiate the service binding deletion process

## Configuration

The Job is a CronJob with a schedule that can be [configured](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/#cron-schedule-syntax) as a value in the [values.yaml](../../resources/keb/values.yaml) file for the chart.
By default, the CronJob is set according to the following schedule:
```yaml  
kyma-environment-broker.serviceBindingCleanup.schedule: "0 2,14 * * *"
```

Use the following environment variables to configure the Job:

| Environment variable         | Description                                                                                                               | Default value                            |
|------------------------------|---------------------------------------------------------------------------------------------------------------------------|------------------------------------------|
| **APP_JOB_DRY_RUN**          | Specifies whether to run the Job in the [`dry-run` mode](#details).                                                       | `true`                                   |
| **APP_JOB_REQUEST_TIMEOUT**  | Specifies the time limit for the DELETE request.                                                                          | `2s`                                     |
| **APP_JOB_REQUEST_RETRIES**  | Specifies the number of retries for the timeouted DELETE request.                                                         | `2`                                      |
| **APP_DATABASE_USER**        | Specifies the username for the database.                                                                                  | `postgres`                               |
| **APP_DATABASE_PASSWORD**    | Specifies the user password for the database.                                                                             | `password`                               |
| **APP_DATABASE_HOST**        | Specifies the host of the database.                                                                                       | `localhost`                              |
| **APP_DATABASE_PORT**        | Specifies the port for the database.                                                                                      | `5432`                                   |
| **APP_DATABASE_NAME**        | Specifies the name of the database.                                                                                       | `provisioner`                            |
| **APP_DATABASE_SSLMODE**     | Activates the SSL mode for PostgreSQL. See [all the possible values](https://www.postgresql.org/docs/9.1/libpq-ssl.html). | `disable`                                |
| **APP_DATABASE_SSLROOTCERT** | Specifies the location of CA cert of PostgreSQL. (Optional)                                                               | None                                     |
| **APP_BROKER_URL**           | Specifies the KEB URL.                                                                                                    | `https://kyma-env-broker.kyma.local`     |
| **APP_BROKER_TOKEN_URL**     | Specifies the KEB OAuth token endpoint.                                                                                   | `https://oauth.2kyma.local/oauth2/token` |
| **APP_BROKER_CLIENT_ID**     | Specifies the username for the OAuth2 authentication in KEB.                                                              | None                                     |
| **APP_BROKER_CLIENT_SECRET** | Specifies the password for the OAuth2 authentication in KEB.                                                              | None                                     |
| **APP_BROKER_SCOPE**         | Specifies the scope for the OAuth2 authentication in KEB.                                                                 | None                                     |

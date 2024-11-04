# Trial Cleanup CronJob and Free Cleanup CronJob

Trial Cleanup CronJob and Free Cleanup CronJob are Jobs that make the SAP BTP, Kyma runtime instances with the `trial` or `free` plans expire 14 or 30 days after their creation, respectively.
Expiration means that the Kyma runtime instance is suspended and the `expired` flag is set.

## Details

For each instance meeting the criteria, a PATCH request is sent to Kyma Environment Broker (KEB). This instance is marked as `expired`, and if it is in the `succeeded` state, the suspension process is started.
If the instance is already in the `suspended` state, this instance is just marked as `expired`.

### Dry-Run Mode

If you need to test the Job, you can run it in the `dry-run` mode.
In that mode, the Job only logs the information about the candidate instances, that is, instances meeting the configured criteria. The instances are not affected.

## Prerequisites

Both Jobs require access to:

* the KEB database to get the IDs of the instances with the `trial` or `free` plan which are not expired yet
* KEB to initiate the Kyma runtime instance suspension

## Configuration

Jobs are CronJobs with a schedule that can be [configured](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/#cron-schedule-syntax) as a value in the [values.yaml](../../resources/keb/values.yaml) file for the chart.
By default, CronJobs are set according to the following schedules:

* Trial Cleanup CronJob runs every day at 1:15 AM:

```yaml  
kyma-environment-broker.trialCleanup.schedule: "15 1 * * *"
```

* Free Cleanup CronJob runs every hour at 40 minutes past the hour:

```yaml
kyma-environment-broker.freeCleanup.schedule: "40 * * * *"
```

Use the following environment variables to configure the Jobs:

| Environment variable         | Description                                                                                                                           | Default value                            |
|------------------------------|---------------------------------------------------------------------------------------------------------------------------------------|------------------------------------------|
| **APP_DRY_RUN**              | Specifies whether to run the Job in the [`dry-run` mode](#details).                                                                   | `true`                                   |
| **APP_EXPIRATION_PERIOD**    | Specifies the [expiration period](#trial-cleanup-cronjob-and-free-cleanup-cronjob) for the instances with the `trial` or `free` plan. | `336h`                                   |
| **APP_DATABASE_USER**        | Specifies the username for the database.                                                                                              | `postgres`                               |
| **APP_DATABASE_PASSWORD**    | Specifies the user password for the database.                                                                                         | `password`                               |
| **APP_DATABASE_HOST**        | Specifies the host of the database.                                                                                                   | `localhost`                              |
| **APP_DATABASE_PORT**        | Specifies the port for the database.                                                                                                  | `5432`                                   |
| **APP_DATABASE_NAME**        | Specifies the name of the database.                                                                                                   | `provisioner`                            |
| **APP_DATABASE_SSLMODE**     | Activates the SSL mode for PostgreSQL. See [all the possible values](https://www.postgresql.org/docs/9.1/libpq-ssl.html).             | `disable`                                |
| **APP_DATABASE_SSLROOTCERT** | Specifies the location of CA cert of PostgreSQL. (Optional)                                                                           | None                                     |
| **APP_BROKER_URL**           | Specifies the KEB URL.                                                                                                                | `https://kyma-env-broker.kyma.local`     |
| **APP_BROKER_TOKEN_URL**     | Specifies the KEB OAuth token endpoint.                                                                                               | `https://oauth.2kyma.local/oauth2/token` |
| **APP_BROKER_CLIENT_ID**     | Specifies the username for the OAuth2 authentication in KEB.                                                                          | None                                     |
| **APP_BROKER_CLIENT_SECRET** | Specifies the password for the OAuth2 authentication in KEB.                                                                          | None                                     |
| **APP_BROKER_SCOPE**         | Specifies the scope for the OAuth2 authentication in KEB.                                                                             | None                                     |

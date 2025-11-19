# Trial Cleanup CronJob and Free Cleanup CronJob

Trial Cleanup CronJob and Free Cleanup CronJob are Jobs that make the SAP BTP, Kyma runtime instances with the trial or free plans expire 14 or 30 days after their creation, respectively.
Expiration means that the Kyma runtime instance is suspended and the `expired` flag is set.

## Details

For each instance meeting the criteria, a PATCH request is sent to Kyma Environment Broker (KEB). This instance is marked as `expired`, and if it is in the `succeeded` state, the suspension process is started.
If the instance is already in the `suspended` state, this instance is just marked as `expired`.

### Dry-Run Mode

If you need to test the Job, you can run it in the `dry-run` mode.
In that mode, the Job only logs the information about the candidate instances, that is, instances meeting the configured criteria. The instances are not affected.

## Prerequisites

* The KEB database to get the IDs of the instances with the trial or free plan which are not expired yet
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
                         |
### Trial Cleanup CronJob

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
| **APP_DRY_RUN** | <code>false</code> | If true, the job only logs what would be deleted without actually removing any data. |
| **APP_EXPIRATION_&#x200b;PERIOD** | <code>336h</code> | Specifies how long a trial instance can exist before being expired. |
| **APP_PLAN_ID** | <code>7d55d31d-35ae-4438-bf13-6ffdfa107d9f</code> | The ID of the trial plan to be used for cleanup. |
| **APP_TEST_RUN** | <code>false</code> | If true, runs the job in test mode. |
| **APP_TEST_SUBACCOUNT_&#x200b;ID** | <code>prow-keb-trial-suspension</code> | Subaccount ID used for test runs. |
| **DATABASE_EMBEDDED** | <code>true</code> | - |


### Free Cleanup CronJob

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
| **APP_DRY_RUN** | <code>false</code> | If true, the job only logs what would be deleted without actually removing any data. |
| **APP_EXPIRATION_&#x200b;PERIOD** | <code>2160h</code> | Specifies how long a free instance can exist before being eligible for cleanup. |
| **APP_PLAN_ID** | <code>b1a5764e-2ea1-4f95-94c0-2b4538b37b55</code> | The ID of the free plan to be used for cleanup. |
| **APP_TEST_RUN** | <code>false</code> | If true, runs the job in test mode (no real deletions, for testing purposes). |
| **APP_TEST_SUBACCOUNT_&#x200b;ID** | <code>prow-keb-trial-suspension</code> | Subaccount ID used for test runs. |
| **DATABASE_EMBEDDED** | <code>true</code> | - |


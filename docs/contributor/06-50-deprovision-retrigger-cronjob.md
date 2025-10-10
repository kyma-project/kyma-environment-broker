# Deprovision Retrigger CronJob

Deprovision Retrigger CronJob is a Job that attempts to deprovision a SAP BTP, Kyma runtime instance once again.

## Details

During regular deprovisioning, you can omit some steps due to the occurrence of some errors. These errors do not cause the deprovisioning process to fail.
You can ignore some not-severe, temporary errors, proceed with deprovisioning and declare the process successful. The not-completed steps
can be retried later. Store the list of not-completed steps, and mark the deprovisioning operation by setting `deletedAt` to the current timestamp.
The Job iterates over the instances, and for each one with `deletedAt` appropriately set, sends a DELETE to Kyma Environment Broker (KEB).  

## Prerequisites

* The KEB database to get the IDs of the instances with not completed steps
* KEB to request Kyma runtime deprovisioning

## Configuration

The Job is a CronJob with a schedule that can be [configured](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/#cron-schedule-syntax) as a value in the [values.yaml](../../resources/keb/values.yaml) file for the chart.
By default, the CronJob is set to run every day at 3:00 am:

```yaml  
kyma-environment-broker.trialCleanup.schedule: "0,15,30,45 * * * *"
```

> [!NOTE]
> If you need to test the Job, you can run it in the `dry-run` mode.
> In this mode, the Job only logs the information about the candidate instances, that is, instances meeting the configured criteria. The instances are not affected.

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
| **APP_DATABASE_&#x200b;TIMEZONE** | None | Specifies the "timezone" parameter in the DB connection URL |
| **APP_DATABASE_USER** | None | Specifies the username for the database. |
| **APP_DRY_RUN** | <code>true</code> | If true, the job runs in dry-run mode and does not actually retrigger deprovisioning. |
| **DATABASE_EMBEDDED** | <code>true</code> | - |

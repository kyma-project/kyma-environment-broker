# Environments Cleanup CronJob

> ### Caution:
> The CronJob must run only in the development environment. You must be connected to the development Kubernetes cluster before applying the CronJob.

> ### Note:
> Environments Cleanup CronJob is applied manually. There is no automatic release process for the Job because of its destructiveness. To get more details on using the Job, visit its [`README`](../../utils/kyma-environments-cleanup-job/README.md).

Environments Cleanup CronJob removes Kyma runtimes which are older than 24h. The CronJob is scheduled to run daily at midnight local time defined in the system.

## Prerequisites

* Gardener project of choice to filter Shoots without a proper label and remove lingering Shoots
* The Kyma Environment Broker (KEB) database to get an Instance ID for each SAP BTP, Kyma runtime marked for deletion
* KEB to trigger Kyma runtime deprovisioning
* Kubernetes client to clean up Runtime custom resources (CRs) created directly using Kyma Infrastructure Manager (KIM) and omitting KEB

## Configuration

The Environments Cleanup binary allows you to override some configuration parameters. You can specify the following environment variables:

| Environment variable | Description | Default value |
|---|---|---|
| **APP_MAX_AGE_HOURS** | Defines the maximum time a Shoot can live without deletion in case the label is not specified. The Shoot age is provided in hours. | `24h` |
| **APP_LABEL_SELECTOR** | Defines the label selector to filter out Shoots for deletion. | `owner.do-not-delete!=true` |
| **APP_GARDENER_PROJECT** | Specifies the name of a Gardener project. | `kyma-dev` |
| **APP_GARDENER_KUBECONFIG_PATH**  | Specifies the kubeconfig path to a Gardener cluster.  | `/gardener/kubeconfig/kubeconfig`  |
| **APP_DATABASE_USER** | Specifies the username for the database. | `postgres` |
| **APP_DATABASE_PASSWORD** | Specifies the user password for the database. | `password` |
| **APP_DATABASE_HOST** | Specifies the host of the database. | `localhost` |
| **APP_DATABASE_PORT** | Specifies the port for the database. | `5432` |
| **APP_DATABASE_NAME** | Specifies the name of the database. | `broker` |
| **APP_DATABASE_SSLMODE** | Activates the SSL mode for PostgrSQL. See [all the possible values](https://www.postgresql.org/docs/9.1/libpq-ssl.html).  | `disable`|
| **APP_DATABASE_SSLROOTCERT** | Specifies the location of CA cert of PostgreSQL. (Optional)  | None |
| **APP_DATABASE_SECRET_KEY** | Database encryption key. (Optional) | None |
| **APP_BROKER_URL**  | Specifies the KEB URL. | `https://kyma-env-broker.kyma.local` |

[![REUSE status](https://api.reuse.software/badge/github.com/kyma-project/kyma-environment-broker)](https://api.reuse.software/info/github.com/kyma-project/kyma-environment-broker)
# Kyma Environment Broker

## Overview

Kyma Environment Broker (KEB) is a component that allows you to provision [SAP BTP, Kyma runtime](https://kyma-project.io/#/?id=kyma-and-sap-btp-kyma-runtime) on clusters provided by third-party providers. In the process, KEB first uses Provisioner to create a cluster. Then, it uses Reconciler and Lifecycle Manager to install Kyma runtime on the cluster.

## Configuration

KEB binary allows you to override some configuration parameters. You can specify the following environment variables:

| Name | Description | Default value |
|-----|---------|:--------:|
| **APP_PORT** | Specifies the port on which the HTTP server listens. | `8080` |
| **APP_PROVISIONER_DEFAULT_GARDENER_SHOOT_PURPOSE** | Specifies the purpose of the created cluster. The possible values are: `development`, `evaluation`, `production`, `testing`. | `development` |
| **APP_PROVISIONER_URL** | Specifies a URL to the Runtime Provisioner's API. | None |
| **APP_DIRECTOR_URL** | Specifies the Director's URL. | `http://compass-director.compass-system.svc.cluster.local:3000/graphql` |
| **APP_DIRECTOR_OAUTH_TOKEN_URL** | Specifies the URL for OAuth authentication. | None |
| **APP_DIRECTOR_OAUTH_CLIENT_ID** | Specifies the client ID for OAuth authentication. | None |
| **APP_DIRECTOR_OAUTH_SCOPE** | Specifies the scopes for OAuth authentication. | `runtime:read runtime:write` |
| **APP_DATABASE_USER** | Defines the database username. | `postgres` |
| **APP_DATABASE_PASSWORD** | Defines the database user password. | `password` |
| **APP_DATABASE_HOST** | Defines the database host. | `localhost` |
| **APP_DATABASE_PORT** | Defines the database port. | `5432` |
| **APP_DATABASE_NAME** | Defines the database name. | `broker` |
| **APP_DATABASE_SSLMODE** | Specifies the SSL Mode for PostgreSQL. See [all the possible values](https://www.postgresql.org/docs/9.1/libpq-ssl.html).  | `disable`|
| **APP_DATABASE_SSLROOTCERT** | Specifies the location of CA cert of PostgreSQL. (Optional)  | None |
| **APP_KYMA_VERSION** | Specifies the default Kyma version. | None |
| **APP_ENABLE_ON_DEMAND_VERSION** | If set to `true`, a user can specify a Kyma version in a provisioning request. | `false` |
| **APP_VERSION_CONFIG_NAMESPACE** | Defines the namespace with the ConfigMap that contains Kyma versions for global accounts configuration. | None |
| **APP_VERSION_CONFIG_NAME** | Defines the name of the ConfigMap that contains Kyma versions for global accounts configuration. | None |
| **APP_PROVISIONER_MACHINE_IMAGE** | Defines the Gardener machine image used in a provisioned node. | None |
| **APP_PROVISIONER__MACHINE_IMAGE_VERSION** | Defines the Gardener image version used in a provisioned cluster. | None |
| **APP_TRIAL_REGION_MAPPING_FILE_PATH** | Defines a path to the file which contains a mapping between the platform region and the Trial plan region. | None |
| **APP_GARDENER_PROJECT** | Defines the project in which the cluster is created. | `kyma-dev` |
| **APP_GARDENER_SHOOT_DOMAIN** | Defines the domain for clusters created in Gardener. | `shoot.canary.k8s-hana.ondemand.com` |
| **APP_GARDENER_KUBECONFIG_PATH** | Defines the path to the kubeconfig file for Gardener. | `/gardener/kubeconfig/kubeconfig` |
| **APP_AVS_REGION_TAG_CLASS_ID** | Specifies the **TagClassId** of the tag that contains Gardener cluster's region. | None |
| **APP_PROFILER_MEMORY** | Enables memory profiling every sampling period with the default location `/tmp/profiler`, backed by a persistent volume. | `false` |

## Read More

To learn more about how to use KEB, read the documentation in the [`user`](./docs/user/) directory.
For more technical details on KEB, go to the [`contributor`](./docs/contributor/) directory.

## Contributing
<!--- mandatory section - do not change this! --->

See the [Contributing](CONTRIBUTING.md) guidelines.

## Code of Conduct
<!--- mandatory section - do not change this! --->

See the [Code of Conduct](CODE_OF_CONDUCT.md) document.

## Licensing
<!--- mandatory section - do not change this! --->

See the [license](./LICENSE) file.

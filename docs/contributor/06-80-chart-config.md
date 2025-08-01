| Parameter | Description | Default Value |
| --- | --- | --- |
| deployment.image.pullPolicy | - | `Always` |
| deployment.replicaCount | - | 1 |
| deployment.securityContext.<br>runAsUser | - | 2000 |
| global.database.cloudsqlproxy.<br>enabled | - | False |
| global.database.cloudsqlproxy.<br>workloadIdentity.enabled | - | False |
| global.database.embedded.<br>enabled | - | True |
| global.database.managedGCP.<br>encryptionSecretName | Name of the Kubernetes Secret containing the encryption | `kcp-storage-client-secret` |
| global.database.managedGCP.<br>encryptionSecretKey | Key in the encryption Secret for the encryption key | `secretKey` |
| global.database.managedGCP.<br>hostSecretKey | Key in the database Secret for the database host | `postgresql-serviceName` |
| global.database.managedGCP.<br>instanceConnectionName | - | `` |
| global.database.managedGCP.<br>nameSecretKey | Key in the database Secret for the database name | `postgresql-broker-db-name` |
| global.database.managedGCP.<br>passwordSecretKey | Key in the database Secret for the database password | `postgresql-broker-password` |
| global.database.managedGCP.<br>portSecretKey | Key in the database Secret for the database port | `postgresql-servicePort` |
| global.database.managedGCP.<br>secretName | Name of the Kubernetes Secret containing DB connection values | `kcp-postgresql` |
| global.database.managedGCP.<br>sslModeSecretKey | Key in the database Secret for the SSL mode | `postgresql-sslMode` |
| global.database.managedGCP.<br>userNameSecretKey | Key in the database Secret for the database user | `postgresql-broker-username` |
| global.images.cloudsql_proxy.<br>repository | - | `eu.gcr.io/sap-ti-dx-kyma-mps-dev/images/cloudsql-proxy` |
| global.images.cloudsql_proxy.<br>tag | - | `2.11.3-sap` |
| global.images.container_registry.<br>path | - | `europe-docker.pkg.dev/kyma-project/prod` |
| global.images.kyma_environment_broker.<br>dir | - | `None` |
| global.images.kyma_environment_broker.<br>version | - | `1.21.17` |
| global.images.kyma_environment_broker_schema_migrator.<br>dir | - | `None` |
| global.images.kyma_environment_broker_schema_migrator.<br>version | - | `1.21.17` |
| global.images.kyma_environments_subaccount_cleanup_job.<br>dir | - | `None` |
| global.images.kyma_environments_subaccount_cleanup_job.<br>version | - | `1.21.17` |
| global.images.kyma_environment_trial_cleanup_job.<br>dir | - | `None` |
| global.images.kyma_environment_trial_cleanup_job.<br>version | - | `1.21.17` |
| global.images.kyma_environment_expirator_job.<br>dir | - | `None` |
| global.images.kyma_environment_expirator_job.<br>version | - | `1.21.17` |
| global.images.kyma_environment_deprovision_retrigger_job.<br>dir | - | `None` |
| global.images.kyma_environment_deprovision_retrigger_job.<br>version | - | `1.21.17` |
| global.images.kyma_environment_runtime_reconciler.<br>dir | - | `None` |
| global.images.kyma_environment_runtime_reconciler.<br>version | - | `1.21.17` |
| global.images.kyma_environment_subaccount_sync.<br>dir | - | `None` |
| global.images.kyma_environment_subaccount_sync.<br>version | - | `1.21.17` |
| global.images.kyma_environment_globalaccounts.<br>dir | - | `None` |
| global.images.kyma_environment_globalaccounts.<br>version | - | `1.21.17` |
| global.images.kyma_environment_service_binding_cleanup_job.<br>dir | - | `None` |
| global.images.kyma_environment_service_binding_cleanup_job.<br>version | - | `1.21.17` |
| global.ingress.domainName | - | `localhost` |
| global.istio.gateway | - | `kyma-system/kyma-gateway` |
| global.istio.proxy.port | - | 15020 |
| global.kyma_environment_broker.<br>serviceAccountName | - | `kcp-kyma-environment-broker` |
| global.secrets.enabled | - | True |
| global.secrets.mechanism | - | `vso` |
| global.secrets.vso.mount | - | `kcp-dev` |
| global.secrets.vso.namespace | - | `kyma` |
| global.secrets.vso.refreshAfter | - | `30s` |
| fullnameOverride | - | `kcp-kyma-environment-broker` |
| host | - | `kyma-env-broker` |
| imagePullSecret | - | `` |
| manageSecrets | If true, this Helm chart will create and manage Kubernetes Secret resources for credentials Set to false if you want to manage these secrets externally or manually, and prevent the chart from creating them | True |
| namePrefix | - | `kcp` |
| nameOverride | - | `kyma-environment-broker` |
| runtimeAllowedPrincipals | We usually recommend not to specify default resources and to leave this as a conscious choice for the user. This also increases chances charts run on environments with little resources, such as Minikube. If you do want to specify resources, uncomment the following lines, adjust them as necessary, and remove the curly braces after 'resources:'. limits: cpu: 100m memory: 128Mi requests: cpu: 100m memory: 128Mi | `- cluster.local/ns/kcp-system/sa/kcp-kyma-metrics-collector` |
| service.port | - | 80 |
| service.type | - | `ClusterIP` |
| swagger.virtualService.<br>enabled | - | True |
| archiving.enabled | If true, enables the archiving mechanism, which stores data about deprovisioned instances in an archive table at the end of the deprovisioning process | False |
| archiving.dryRun | If true, runs the archiving process in dry-run mode: Makes no changes to the database, only logs what is to be archived or deleted | True |
| broker.allowUpdateExpiredInstanceWithContext | Allow update of expired instance | `false` |
| broker.binding.bindablePlans | Comma-separated list of plan names for which service binding is enabled, for example, "aws,gcp" | `aws` |
| broker.binding.createBindingTimeout | Timeout for creating a binding, for example, 15s, 1m | `15s` |
| broker.binding.enabled | Enables or disables the service binding endpoint (true/false) | False |
| broker.binding.expirationSeconds | Default expiration time (in seconds) for a binding if not specified in the request | 600 |
| broker.binding.maxBindingsCount | Maximum number of non-expired bindings allowed per instance | 10 |
| broker.binding.maxExpirationSeconds | Maximum allowed expiration time (in seconds) for a binding | 7200 |
| broker.binding.minExpirationSeconds | Minimum allowed expiration time (in seconds) for a binding. Can't be lower than 600 seconds. Forced by Gardener | 600 |
| broker.defaultRequestRegion | Default platform region for requests if not specified | `cf-eu10` |
| broker.disableSapConvergedCloud | If true, disables the SAP Cloud Infrastructure plan in the KEB. When set to true, users cannot provision SAP Cloud Infrastructure clusters | False |
| broker.enableJwks | If true, enables the handling of the encoded JWKS array, temporary feature flag | `false` |
| broker.enablePlans | Comma-separated list of plan names enabled and available for provisioning in KEB | `azure,gcp,azure_lite,trial,aws` |
| broker.enablePlanUpgrades | If true, allows users to upgrade their plans (if a plan supports upgrades) | `false` |
| broker.enableShootAndSeedSameRegion | If true, enforces that the Gardener seed is placed in the same region as the shoot region selected during provisioning | `false` |
| broker.freeDocsURL | URL to the documentation of free Kyma runtimes. Used in API responses and UI labels to direct users to help or documentation about free plans | `https://help.sap.com/docs/` |
| broker.freeExpirationPeriod | Determines when to show expiration info to users | `720h` |
| broker.gardenerSeedsCache | Name of the Kubernetes ConfigMap used as a cache for Gardener seeds | `gardener-seeds-cache` |
| broker.includeAdditionalParamsInSchema | If true, additional (advanced or less common) parameters are included in the provisioning schema for service plans | `false` |
| broker.monitorAdditionalProperties | If true, collects properties from the provisioning request that are not explicitly defined in the schema and stores them in persistent storage | False |
| broker.onlyOneFreePerGA | If true, restricts each global account to only one free (freemium) Kyma runtime. When enabled, provisioning another free environment for the same global account is blocked even if the previous one is deprovisioned | `false` |
| broker.onlySingleTrialPerGA | If true, restricts each global account to only one active trial Kyma runtime at a time When enabled, provisioning another trial environment for the same global account is blocked until the previous one is deprovisioned | `true` |
| broker.operationTimeout | Maximum allowed duration for processing a single operation (provisioning, deprovisioning, etc.) If the operation exceeds this timeout, it is marked as failed. Example: "7h" for 7 hours | `7h` |
| broker.port | Port for the broker HTTP server | `8080` |
| broker.rejectUnsupportedParameters | If true, rejects requests that contain parameters that are not defined in schemas | `false` |
| broker.showFreeExpirationInfo | If true, adds expiration information for free plan Kyma runtimes to API responses and UI labels | `false` |
| broker.showTrialExpirationInfo | If true, adds expiration information for trial plan Kyma runtimes to API responses and UI labels | `false` |
| broker.statusPort | Port for the broker status/health endpoint | `8071` |
| broker.subaccountMovementEnabled | If true, enables subaccount movement (allows changing global account for an instance) | `false` |
| broker.subaccountsIdsToShowTrialExpirationInfo | Shows trial expiration information for specific subaccounts in the UI and API responses | `a45be5d8-eddc-4001-91cf-48cc644d571f` |
| broker.trialDocsURL | URL to the documentation for trial Kyma runtimes. Used in API responses and UI labels | `https://help.sap.com/docs/` |
| broker.updateCustomResourcesLabelsOnAccountMove | If true, updates runtimeCR labels when moving subaccounts | `false` |
| broker.useAdditionalOIDCSchema | If true, enables the new list-based OIDC schema, allowing multiple OIDC configurations for a runtime | `false` |
| provisioning.maxStepProcessingTime | Maximum time a worker is allowed to process a step before it must return to the provisioning queue | `2m` |
| provisioning.workersAmount | Number of workers in provisioning queue | 20 |
| update.maxStepProcessingTime | Maximum time a worker is allowed to process a step before it must return to the update queue | `2m` |
| update.workersAmount | Number of workers in update queue | 20 |
| deprovisioning.maxStepProcessingTime | Maximum time a worker is allowed to process a step before it must return to the deprovisioning queue | `2m` |
| deprovisioning.workersAmount | Number of workers in deprovisioning queue | 20 |
| cleaning.dryRun | If true, the cleaning process runs in dry-run mode and does not actually delete any data from the database | True |
| cleaning.enabled | If true, enables the cleaning process, which removes all data about deprovisioned instances from the database | False |
| configPaths.catalog | Path to the service catalog configuration file | `/config/catalog.yaml` |
| configPaths.freemiumWhitelistedGlobalAccountIds | Path to the list of global account IDs that are allowed unlimited access to freemium (free) Kyma runtimes. Only accounts listed here can provision more than the default limit of free environments | `/config/freemiumWhitelistedGlobalAccountIds.yaml` |
| configPaths.hapRule | Path to the rules for mapping plans and regions to hyperscaler account pools | `/config/hapRule.yaml` |
| configPaths.plansConfig | Path to the plans configuration file, which defines available service plans | `/config/plansConfig.yaml` |
| configPaths.providersConfig | Path to the providers configuration file, which defines hyperscaler/provider settings | `/config/providersConfig.yaml` |
| configPaths.quotaWhitelistedSubaccountIds | Path to the list of subaccount IDs that are allowed to bypass quota restrictions | `/config/quotaWhitelistedSubaccountIds.yaml` |
| configPaths.regionsSupportingMachine | Path to the list of regions that support machine-type selection | `/config/regionsSupportingMachine.yaml` |
| configPaths.skrDNSProvidersValues | Path to the DNS providers values | `/config/skrDNSProvidersValues.yaml` |
| configPaths.skrOIDCDefaultValues | Path to the default OIDC values | `/config/skrOIDCDefaultValues.yaml` |
| configPaths.trialRegionMapping | Path to the region mapping for trial environments | `/config/trialRegionMapping.yaml` |
| configPaths.cloudsqlSSLRootCert | Path to the Cloud SQL SSL root certificate file | `/secrets/cloudsql-sslrootcert/server-ca.pem` |
| disableProcessOperationsInProgress | If true, the broker does NOT resume processing operations (provisioning, deprovisioning, updating, etc.) that were in progress when the broker process last stopped or restarted | `false` |
| edp.adminURL | Base URL for the EDP admin API | `TBD` |
| edp.authURL | OAuth2 token endpoint for EDP | `TBD` |
| edp.disabled | If true, disables EDP integration | True |
| edp.environment | EDP environment, for example, dev, prod | `dev` |
| edp.namespace | EDP namespace to use | `kyma-dev` |
| edp.required | If true, EDP integration is required | False |
| edp.secret | OAuth2 client secret for EDP | `TBD` |
| edp.secretName | Name of the Kubernetes Secret containing EDP credentials | `edp-creds` |
| edp.secretKey | OAuth2 client secret key name used to fetch the EDP secret from the Kubernetes Secret | `secret` |
| events.enabled | Enables or disables the /events API and event storage for operation events (true/false) | True |
| freemiumWhitelistedGlobalAccountIds | List of global account IDs that are allowed unlimited access to freemium (free) Kyma runtimes Only accounts listed here can provision more than the default limit of free environments | `whitelist:` |
| gardener.kubeconfigPath | Path to the kubeconfig file for accessing the Gardener cluster | `/gardener/kubeconfig/kubeconfig` |
| gardener.project | Gardener project connected to SA for HAP credentials lookup | `kyma-dev` |
| gardener.secretName | Name of the Kubernetes Secret containing Gardener credentials | `gardener-credentials` |
| gardener.shootDomain | Default domain for shoots (clusters) created by Gardener | `kyma-dev.shoot.canary.k8s-hana.ondemand.com` |
| infrastructureManager.<br>controlPlaneFailureTolerance | Sets the failure tolerance level for the Kubernetes control plane in Gardener clusters Possible values: empty (default), "node", or "zone" | `` |
| infrastructureManager.<br>defaultShootPurpose | Sets the default purpose for Gardener shoots (clusters) created by the broker Possible values: development, evaluation, production, testing | `development` |
| infrastructureManager.<br>defaultTrialProvider | Sets the default cloud provider for trial Kyma runtimes, for example, Azure, AWS | `Azure` |
| infrastructureManager.<br>ingressFilteringPlans | Comma-separated list of plan names for which ingress filtering is available | `azure,gcp,aws` |
| infrastructureManager.<br>kubernetesVersion | Sets the default Kubernetes version for new clusters provisioned by the broker | `1.16.9` |
| infrastructureManager.<br>machineImage | Sets the default machine image name for nodes in provisioned clusters. If empty, the Gardener default value is used | `` |
| infrastructureManager.<br>machineImageVersion | Sets the version of the machine image for nodes in provisioned clusters. If empty, the Gardener default value is used | `` |
| infrastructureManager.<br>multiZoneCluster | If true, enables provisioning of clusters with nodes distributed across multiple availability zones | `false` |
| infrastructureManager.<br>useSmallerMachineTypes | If true, provisions trial, freemium, and azure_lite clusters using smaller machine types | `false` |
| kubeconfig.allowOrigins | Specifies which origins are allowed for Cross-Origin Resource Sharing (CORS) on the /kubeconfig endpoint | `*` |
| kymaDashboardConfig.landscapeURL | The base URL of the Kyma Dashboard used to generate links to the web UI for Kyma runtimes | `https://dashboard.dev.kyma.cloud.sap` |
| lifecycleManager.disabled | When disabled, the broker does not create, update, or delete the KymaCR | `false` |
| metricsv2.enabled | If true, enables metricsv2 collection and Prometheus exposure | False |
| metricsv2.operationResultFinishedOperationRetentionPeriod | Duration of retaining finished operation results in memory | `3h` |
| metricsv2.operationResultPollingInterval | Frequency of polling for operation results | `1m` |
| metricsv2.operationResultRetentionPeriod | Duration of retaining operation results | `1h` |
| metricsv2.operationStatsPollingInterval | Frequency of polling for operation statistics | `1m` |
| multipleContexts | If true, generates kubeconfig files with multiple contexts (if possible) instead of a single context | False |
| profiler.memory | Enables memory profiler (true/false) | False |
| quotaLimitCheck.enabled | If true, validates during provisioning that the assigned quota for the subaccount is not exceeded | False |
| quotaLimitCheck.interval | The interval between requests to the Entitlements API in case of errors | `1s` |
| quotaLimitCheck.retries | The number of retry attempts made when the Entitlements API request fails | 5 |
| quotaWhitelistedSubaccountIds | List of subaccount IDs that have unlimited quota for Kyma runtimes. Only subaccounts listed here can provision beyond their assigned quota limits | `whitelist:` |
| regionsSupportingMachine | Defines which machine type families are available in which regions (and optionally, zones) Restricts provisioning of listed machine types to the specified regions/zones only If a machine type is not listed, it is considered available in all regions | `` |
| runtimeConfiguration | Defines the default KymaCR template. | `default: \|-<br><br>  kyma-template: \|-<br><br>    apiVersion: operator.kyma-project.io/v1beta2<br><br>    kind: Kyma<br><br>    metadata:<br><br>      labels:<br><br>        "operator.kyma-project.io/managed-by": "lifecycle-manager"<br><br>      name: tbd<br><br>      namespace: kcp-system<br><br>    spec:<br><br>      channel: fast<br><br>      modules: []<br><br>  additional-components: []` |
| skrDNSProvidersValues | Contains DNS provider configuration for SKR clusters | `providers: []` |
| skrOIDCDefaultValues | Contains the default OIDC configuration for SKR clusters | `clientID: "9bd05ed7-a930-44e6-8c79-e6defeb7dec9"<br><br>groupsClaim: "groups"<br><br>groupsPrefix: "-"<br><br>issuerURL: "https://kymatest.accounts400.ondemand.com"<br><br>signingAlgs: [ "RS256" ]<br><br>usernameClaim: "sub"<br><br>usernamePrefix: "-"` |
| stepTimeouts.checkRuntimeResourceCreate | Maximum time to wait for a runtime resource to be created before considering the step as failed | `60m` |
| stepTimeouts.checkRuntimeResourceDeletion | Maximum time to wait for a runtime resource to be deleted before considering the step as failed | `60m` |
| stepTimeouts.checkRuntimeResourceUpdate | Maximum time to wait for a runtime resource to be updated before considering the step as failed | `180m` |
| testConfig.kebDeployment.<br>useAnnotations | - | False |
| testConfig.kebDeployment.<br>weight | - | `2` |
| trialRegionsMapping | Determines the Kyma region for a trial environment based on the requested platform region | `cf-eu10: europe<br><br>cf-us10: us<br><br>cf-ap21: asia` |
| osbUpdateProcessingEnabled | If true, the broker processes update requests for service instances | `true` |
| cis.accounts.authURL | The OAuth2 token endpoint (authorization URL) used to obtain access tokens for authenticating requests to the CIS Accounts API | `TBD` |
| cis.accounts.id | The OAuth2 client ID used for authenticating requests to the CIS Accounts API | `TBD` |
| cis.accounts.secret | The OAuth2 client secret used together with the client ID for authentication with the CIS Accounts API | `TBD` |
| cis.accounts.secretName | The name of the Kubernetes Secret containing the CIS Accounts client ID and secret | `cis-creds-accounts` |
| cis.accounts.serviceURL | The base URL of the CIS Accounts API endpoint, used for fetching subaccount data | `TBD` |
| cis.accounts.clientIdKey | The key in the Kubernetes Secret that contains the CIS v2 client ID | `id` |
| cis.accounts.secretKey | The key in the Kubernetes Secret that contains the CIS v2 client secret | `secret` |
| cis.v1.authURL | The OAuth2 token endpoint (authorization URL) for CIS v1, used to obtain access tokens for authenticating requests | `TBD` |
| cis.v1.eventServiceURL | The endpoint URL for the CIS v1 event service, used to fetch subaccount events | `TBD` |
| cis.v1.id | The OAuth2 client ID used for authenticating requests to the CIS v1 API | `TBD` |
| cis.v1.secret | The OAuth2 client secret used together with the client ID for authentication with the CIS v1 API | `TBD` |
| cis.v1.secretName | The name of the Kubernetes Secret containing the CIS v1 client ID and secret | `cis-creds-v1` |
| cis.v1.clientIdKey | The key in the Kubernetes Secret that contains the CIS v2 client ID | `id` |
| cis.v1.secretKey | The key in the Kubernetes Secret that contains the CIS v2 client secret | `secret` |
| cis.v2.authURL | The OAuth2 token endpoint (authorization URL) for CIS v2, used to obtain access tokens for authenticating requests | `TBD` |
| cis.v2.eventServiceURL | The endpoint URL for the CIS v2 event service, used to fetch subaccount events | `TBD` |
| cis.v2.id | The OAuth2 client ID used for authenticating requests to the CIS v2 API | `TBD` |
| cis.v2.secret | The OAuth2 client secret used together with the client ID for authentication with the CIS v2 API | `TBD` |
| cis.v2.secretName | The name of the Kubernetes Secret containing the CIS v2 client ID and secret | `cis-creds-v2` |
| cis.v2.jobRetries | The number of times a job should be retried in case of failure | 6 |
| cis.v2.maxRequestRetries | The maximum number of request retries to the CIS v2 API in case of errors | `3` |
| cis.v2.rateLimitingInterval | The minimum interval between requests to the CIS v2 API in case of errors | `2s` |
| cis.v2.requestInterval | The interval between requests to the CIS v2 API | `200ms` |
| cis.v2.clientIdKey | The key in the Kubernetes Secret that contains the CIS v2 client ID | `id` |
| cis.v2.secretKey | The key in the Kubernetes Secret that contains the CIS v2 client secret | `secret` |
| cis.entitlements.authURL | The OAuth2 token endpoint (authorization URL) used to obtain access tokens for authenticating requests to the CIS Entitlements API | `TBD` |
| cis.entitlements.id | The OAuth2 client ID used for authenticating requests to the CIS Entitlements API | `TBD` |
| cis.entitlements.secret | The OAuth2 client secret used together with the client ID for authentication with the CIS Entitlements API | `TBD` |
| cis.entitlements.secretName | The name of the Kubernetes Secret containing the CIS Entitlements client ID and secret | `cis-creds-entitlements` |
| cis.entitlements.serviceURL | The base URL of the CIS Entitlements API endpoint, used for fetching quota assignments | `TBD` |
| cis.entitlements.clientIdKey | The key in the Kubernetes Secret that contains the CIS Entitlements client ID | `id` |
| cis.entitlements.secretKey | The key in the Kubernetes Secret that contains the CIS Entitlements client secret | `secret` |
| deprovisionRetrigger.<br>dryRun | If true, the job runs in dry-run mode and does not actually retrigger deprovisioning | True |
| deprovisionRetrigger.<br>enabled | If true, enables the Deprovision Retrigger CronJob, which periodically attempts to deprovision instances that were not fully deleted | True |
| deprovisionRetrigger.<br>schedule | - | `0 2 * * *` |
| freeCleanup.dryRun | If true, the job only logs what would be deleted without actually removing any data | True |
| freeCleanup.enabled | If true, enables the Free Cleanup CronJob | True |
| freeCleanup.expirationPeriod | Specifies how long a free instance can exist before being eligible for cleanup (e.g., 2160h = 90 days) | `2160h` |
| freeCleanup.planID | The ID of the free plan to be used for cleanup | `b1a5764e-2ea1-4f95-94c0-2b4538b37b55` |
| freeCleanup.schedule | - | `0,15,30,45 * * * *` |
| freeCleanup.testRun | If true, runs the job in test mode (no real deletions, for testing purposes) | False |
| freeCleanup.testSubaccountID | Subaccount ID used for test runs | `prow-keb-trial-suspension` |
| globalaccounts.dryRun | If true, runs the global accounts synchronization job in dry-run mode (no changes are made) | True |
| globalaccounts.enabled | If true, enables the global accounts synchronization job | False |
| globalaccounts.name | Name of the global accounts synchronization job or deployment | `kyma-environment-globalaccounts` |
| migratorJobs.argosync.<br>enabled | If true, enables the ArgoCD sync job for schema migration | False |
| migratorJobs.argosync.<br>syncwave | The sync wave value for ArgoCD hooks | `0` |
| migratorJobs.direction | Defines the direction of the schema migration, either "up" or "down" | `up` |
| migratorJobs.enabled | If true, enables all migrator jobs | True |
| migratorJobs.helmhook.<br>enabled | If true, enables the Helm hook job for schema migration | True |
| migratorJobs.helmhook.<br>weight | The weight value for the Helm hook | `1` |
| oidc.groups.admin | - | `runtimeAdmin` |
| oidc.groups.operator | - | `runtimeOperator` |
| oidc.groups.orchestrations | - | `orchestrationsAdmin` |
| oidc.groups.viewer | - | `runtimeViewer` |
| oidc.issuer | - | `https://kymatest.accounts400.ondemand.com` |
| oidc.keysURL | - | `https://kymatest.accounts400.ondemand.com/oauth2/certs` |
| runtimeReconciler.dryRun | If true, runs the reconciler in dry-run mode (no changes are made, only logs actions) | True |
| runtimeReconciler.enabled | Enables or disables the Runtime Reconciler deployment | False |
| runtimeReconciler.jobEnabled | If true, enables the periodic reconciliation job | False |
| runtimeReconciler.jobInterval | Interval (in minutes) between reconciliation job runs | 1440 |
| runtimeReconciler.jobReconciliationDelay | Delay before starting reconciliation after job trigger (e.g., "1s") | `1s` |
| runtimeReconciler.metricsPort | Port on which the reconciler exposes Prometheus metrics | 8081 |
| serviceBindingCleanup.<br>dryRun | If true, the job only logs what would be deleted without actually removing any bindings | True |
| serviceBindingCleanup.<br>enabled | If true, enables the Service Binding Cleanup CronJob | False |
| serviceBindingCleanup.<br>requestRetries | Number of times to retry a failed DELETE request for a binding | 2 |
| serviceBindingCleanup.<br>requestTimeout | Timeout for each DELETE request to the broker | `2s` |
| serviceBindingCleanup.<br>schedule | - | `0 2,14 * * *` |
| subaccountCleanup.enabled | - | `false` |
| subaccountCleanup.nameV1 | - | `kcp-subaccount-cleaner-v1.0` |
| subaccountCleanup.nameV2 | - | `kcp-subaccount-cleaner-v2.0` |
| subaccountCleanup.schedule | - | `0 1 * * *` |
| subaccountCleanup.clientV1VersionName | Client version | `v1.0` |
| subaccountCleanup.clientV2VersionName | Client version | `v2.0` |
| subaccountSync.accountSyncInterval | Interval between full account synchronization runs | `24h` |
| subaccountSync.alwaysSubaccountFromDatabase | If true, fetches subaccountID from the database only when the subaccount is empty | False |
| subaccountSync.cisRateLimits.<br>accounts.maxRequestsPerInterval | Maximum number of requests per interval to the CIS Accounts API | 5 |
| subaccountSync.cisRateLimits.<br>accounts.rateLimitingInterval | Minimum interval between requests to the CIS Accounts API | `2s` |
| subaccountSync.cisRateLimits.<br>events.maxRequestsPerInterval | Maximum number of requests per interval to the CIS Events API | 5 |
| subaccountSync.cisRateLimits.<br>events.rateLimitingInterval | Minimum interval between requests to the CIS Events API | `2s` |
| subaccountSync.enabled | If true, enables the subaccount synchronization job | True |
| subaccountSync.eventsWindowInterval | Time window for collecting events from CIS | `15m` |
| subaccountSync.eventsWindowSize | Size of the time window for collecting events from CIS | `20m` |
| subaccountSync.logLevel | Log level for the subaccount sync job | `info` |
| subaccountSync.metricsPort | Port on which the subaccount sync service exposes Prometheus metrics | 8081 |
| subaccountSync.name | Name of the subaccount sync deployment | `subaccount-sync` |
| subaccountSync.queueSleepInterval | Interval between queue processing cycles | `30s` |
| subaccountSync.storageSyncInterval | Interval between storage synchronization | `5m` |
| subaccountSync.updateResources | If true, enables updating resources during subaccount sync | False |
| trialCleanup.dryRun | If true, the job only logs what would be deleted without actually removing any data | True |
| trialCleanup.enabled | If true, enables the Trial Cleanup CronJob, which removes expired trial Kyma runtimes | True |
| trialCleanup.expirationPeriod | Specifies how long a trial instance can exist before being expired | `336h` |
| trialCleanup.planID | The ID of the trial plan to be used for cleanup | `7d55d31d-35ae-4438-bf13-6ffdfa107d9f` |
| trialCleanup.schedule | - | `15 1 * * *` |
| trialCleanup.testRun | If true, runs the job in test mode | False |
| trialCleanup.testSubaccountID | Subaccount ID used for test runs | `prow-keb-trial-suspension` |
| serviceMonitor.enabled | - | False |
| serviceMonitor.interval | - | `30s` |
| serviceMonitor.scrapeTimeout | - | `10s` |
| vmscrapes.enabled | - | True |
| vmscrapes.interval | - | `30s` |
| vmscrapes.scrapeTimeout | - | `10s` |
| vsoSecrets.secrets.edp.<br>path | - | `edp` |
| vsoSecrets.secrets.edp.<br>secretName | - | `{{ .Values.edp.secretName }}` |
| vsoSecrets.secrets.edp.<br>labels | - | `{{ template "kyma-env-broker.labels" . }}` |
| vsoSecrets.secrets.edp.<br>templating.enabled | - | True |
| vsoSecrets.secrets.edp.<br>templating.keys.secret | - | `keb_edp_secret` |
| vsoSecrets.secrets.cis-v1.<br>path | - | `cis` |
| vsoSecrets.secrets.cis-v1.<br>secretName | - | `{{ .Values.cis.v1.secretName \| required "please specify .Values.cis.v1.secretName"}}` |
| vsoSecrets.secrets.cis-v1.<br>labels | - | `{{ template "kyma-env-broker.labels" . }}` |
| vsoSecrets.secrets.cis-v1.<br>templating.enabled | - | True |
| vsoSecrets.secrets.cis-v1.<br>templating.keys.id | - | `v1_id` |
| vsoSecrets.secrets.cis-v1.<br>templating.keys.secret | - | `v1_secret` |
| vsoSecrets.secrets.cis-v2.<br>path | - | `cis` |
| vsoSecrets.secrets.cis-v2.<br>secretName | - | `{{ .Values.cis.v2.secretName \| required "please specify .Values.cis.v2.secretName"}}` |
| vsoSecrets.secrets.cis-v2.<br>labels | - | `{{ template "kyma-env-broker.labels" . }}` |
| vsoSecrets.secrets.cis-v2.<br>templating.enabled | - | True |
| vsoSecrets.secrets.cis-v2.<br>templating.keys.id | - | `v2_id` |
| vsoSecrets.secrets.cis-v2.<br>templating.keys.secret | - | `v2_secret` |
| vsoSecrets.secrets.cis-accounts.<br>path | - | `cis` |
| vsoSecrets.secrets.cis-accounts.<br>secretName | - | `{{ .Values.cis.accounts.secretName \| required "please specify .Values.cis.accounts.secretName"}}` |
| vsoSecrets.secrets.cis-accounts.<br>labels | - | `{{ template "kyma-env-broker.labels" . }}` |
| vsoSecrets.secrets.cis-accounts.<br>templating.enabled | - | True |
| vsoSecrets.secrets.cis-accounts.<br>templating.keys.id | - | `account_id` |
| vsoSecrets.secrets.cis-accounts.<br>templating.keys.secret | - | `account_secret` |
| vsoSecrets.secrets.cis-entitlements.<br>path | - | `cis` |
| vsoSecrets.secrets.cis-entitlements.<br>secretName | - | `{{ .Values.cis.entitlements.secretName \| required "please specify .Values.cis.entitlements.secretName"}}` |
| vsoSecrets.secrets.cis-entitlements.<br>labels | - | `{{ template "kyma-env-broker.labels" . }}` |
| vsoSecrets.secrets.cis-entitlements.<br>templating.enabled | - | True |
| vsoSecrets.secrets.cis-entitlements.<br>templating.keys.id | - | `entitlements_id` |
| vsoSecrets.secrets.cis-entitlements.<br>templating.keys.secret | - | `entitlements_secret` |

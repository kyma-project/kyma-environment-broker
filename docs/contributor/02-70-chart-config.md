| Parameter | Description | Default Value |
| --- | --- | --- |
| deployment.image.<br>pullPolicy | - | `Always` |
| deployment.<br>replicaCount | - | `1` |
| deployment.securityContext.<br>runAsUser | - | `2000` |
| global.database.cloudsqlproxy.<br>enabled | - | `False` |
| global.database.cloudsqlproxy.<br>workloadIdentity.<br>enabled | - | `False` |
| global.database.embedded.<br>enabled | - | `True` |
| global.database.managedGCP.<br>encryptionSecretName | Name of the Kubernetes Secret containing the encryption. | `kcp-storage-client-secret` |
| global.database.managedGCP.<br>encryptionSecretKey | Key in the encryption Secret for the encryption key. | `secretKey` |
| global.database.managedGCP.<br>hostSecretKey | Key in the database Secret for the database host. | `postgresql-serviceName` |
| global.database.managedGCP.<br>instanceConnectionName | - | `` |
| global.database.managedGCP.<br>nameSecretKey | Key in the database Secret for the database name. | `postgresql-broker-db-name` |
| global.database.managedGCP.<br>passwordSecretKey | Key in the database Secret for the database password. | `postgresql-broker-password` |
| global.database.managedGCP.<br>portSecretKey | Key in the database Secret for the database port. | `postgresql-servicePort` |
| global.database.managedGCP.<br>secretName | Name of the Kubernetes Secret containing DB connection values. | `kcp-postgresql` |
| global.database.managedGCP.<br>sslModeSecretKey | Key in the database Secret for the SSL mode. | `postgresql-sslMode` |
| global.database.managedGCP.<br>userNameSecretKey | Key in the database Secret for the database user. | `postgresql-broker-username` |
| global.images.cloudsql_<br>proxy.repository | - | `eu.gcr.io/sap-ti-dx-kyma-mps-dev/images/cloudsql-proxy` |
| global.images.cloudsql_<br>proxy.tag | - | `2.11.3-sap` |
| global.images.container_<br>registry.path | - | `europe-docker.pkg.dev/kyma-project/prod` |
| global.images.kyma_environment_<br>broker.dir | - | None |
| global.images.kyma_environment_<br>broker.version | - | `1.24.2` |
| global.images.kyma_environment_<br>broker_schema_migrator.<br>dir | - | None |
| global.images.kyma_environment_<br>broker_schema_migrator.<br>version | - | `1.24.2` |
| global.images.kyma_environments_<br>subaccount_cleanup_job.<br>dir | - | None |
| global.images.kyma_environments_<br>subaccount_cleanup_job.<br>version | - | `1.24.2` |
| global.images.kyma_environment_<br>expirator_job.dir | - | None |
| global.images.kyma_environment_<br>expirator_job.<br>version | - | `1.24.2` |
| global.images.kyma_environment_<br>deprovision_retrigger_<br>job.dir | - | None |
| global.images.kyma_environment_<br>deprovision_retrigger_<br>job.version | - | `1.24.2` |
| global.images.kyma_environment_<br>runtime_reconciler.<br>dir | - | None |
| global.images.kyma_environment_<br>runtime_reconciler.<br>version | - | `1.24.2` |
| global.images.kyma_environment_<br>subaccount_sync.dir | - | None |
| global.images.kyma_environment_<br>subaccount_sync.<br>version | - | `1.24.2` |
| global.images.kyma_environment_<br>globalaccounts.dir | - | None |
| global.images.kyma_environment_<br>globalaccounts.<br>version | - | `1.24.2` |
| global.images.kyma_environment_<br>service_binding_cleanup_<br>job.dir | - | None |
| global.images.kyma_environment_<br>service_binding_cleanup_<br>job.version | - | `1.24.2` |
| global.ingress.<br>domainName | - | `localhost` |
| global.istio.gateway | - | `kyma-system/kyma-gateway` |
| global.istio.proxy.<br>port | - | `15020` |
| global.kyma_environment_<br>broker.<br>serviceAccountName | - | `kcp-kyma-environment-broker` |
| global.secrets.<br>enabled | - | `True` |
| global.secrets.<br>mechanism | - | `vso` |
| global.secrets.vso.<br>mount | - | `kcp-dev` |
| global.secrets.vso.<br>namespace | - | `kyma` |
| global.secrets.vso.<br>refreshAfter | - | `30s` |
| fullnameOverride | - | `kcp-kyma-environment-broker` |
| host | - | `kyma-env-broker` |
| imagePullSecret | Specifies the name of the Kubernetes Secret to use for pulling container images from private registries. | `` |
| manageSecrets | If true, this Helm chart creates and manages Kubernetes Secret resources for credentials. Set to false if you want to manage these Secrets externally or manually, and prevent the chart from creating them. | `True` |
| namePrefix | - | `kcp` |
| nameOverride | - | `kyma-environment-broker` |
| useHAPForDeprovisioning | If true, uses HAP for deprovisioning. | `False` |
| runtimeAllowedPrincipals | - | `- cluster.local/ns/kcp-system/sa/kcp-kyma-metrics-collector` |
| service.port | - | `80` |
| service.type | - | `ClusterIP` |
| swagger.virtualService.<br>enabled | - | `True` |
| broker.binding.<br>bindablePlans | Comma-separated list of plan names for which service binding is enabled, for example, "aws,gcp". | `aws` |
| broker.binding.<br>createBindingTimeout | Timeout for creating a binding, for example, 15s, 1m. | `15s` |
| broker.binding.<br>enabled | Enables or disables the service binding endpoint (true/false). | `False` |
| broker.binding.<br>expirationSeconds | Default expiration time (in seconds) for a binding if not specified in the request. | `600` |
| broker.binding.<br>maxBindingsCount | Maximum number of non-expired bindings allowed per instance. | `10` |
| broker.binding.<br>maxExpirationSeconds | Maximum allowed expiration time (in seconds) for a binding. | `7200` |
| broker.binding.<br>minExpirationSeconds | Minimum allowed expiration time (in seconds) for a binding. Can't be lower than 600 seconds. Forced by Gardener. | `600` |
| broker.<br>defaultRequestRegion | Default platform region for requests if not specified. | `cf-eu10` |
| broker.enablePlans | Comma-separated list of plan names enabled and available for provisioning in KEB. | `azure,gcp,azure_lite,trial,aws` |
| broker.<br>enablePlanUpgrades | If true, allows users to upgrade their plans (if a plan supports upgrades). | `false` |
| broker.freeDocsURL | URL to the documentation of free Kyma runtimes. Used in API responses and UI labels to direct users to help or documentation about free plans | `https://help.sap.com/docs/btp/sap-business-technology-platform/using-free-service-plans?version=Cloud` |
| broker.<br>freeExpirationPeriod | Determines when to show expiration info to users. | `720h` |
| broker.<br>gardenerSeedsCache | Name of the Kubernetes ConfigMap used as a cache for Gardener seeds. | `gardener-seeds-cache` |
| broker.<br>monitorAdditionalProperties | If true, collects properties from the provisioning request that are not explicitly defined in the schema and stores them in persistent storage. | `False` |
| broker.<br>onlyOneFreePerGA | If true, restricts each global account to only one freemium (free) Kyma runtime. When enabled, provisioning another free environment for the same global account is blocked even if the previous one is deprovisioned. | `false` |
| broker.<br>onlySingleTrialPerGA | If true, restricts each global account to only one active trial Kyma runtime at a time. When enabled, provisioning another trial environment for the same global account is blocked until the previous one is deprovisioned. | `true` |
| broker.<br>operationTimeout | Maximum allowed duration for processing a single operation (provisioning, deprovisioning, etc.). If the operation exceeds this timeout, it is marked as failed. | `7h` |
| broker.port | Port for the broker HTTP server. | `8080` |
| broker.<br>rejectUnsupportedParameters | If true, rejects requests that contain parameters that are not defined in schemas. | `false` |
| broker.statusPort | Port for the broker status/health endpoint. | `8071` |
| broker.<br>subaccountMovementEnabled | If true, enables subaccount movement (allows changing global account for an instance). | `false` |
| broker.trialDocsURL | URL to the documentation for trial Kyma runtimes. Used in API responses and UI labels. | `https://help.sap.com/docs/` |
| broker.<br>dualStackDocsURL | URL to the documentation for dual-stack networking. Used in dual-stack configuration description in schema. | `https://help.sap.com/docs/btp/sap-business-technology-platform/kyma-runtime-with-dual-stack-support` |
| broker.<br>updateCustomResourcesLabelsOnAccountMove | If true, updates runtimeCR labels when moving subaccounts. | `false` |
| provisioning.<br>maxStepProcessingTime | Maximum time a worker is allowed to process a step before it must return to the provisioning queue. | `2m` |
| provisioning.<br>workersAmount | Number of workers in provisioning queue. | `20` |
| update.<br>maxStepProcessingTime | Maximum time a worker is allowed to process a step before it must return to the update queue. | `2m` |
| update.workersAmount | Number of workers in update queue. | `20` |
| deprovisioning.<br>maxStepProcessingTime | Maximum time a worker is allowed to process a step before it must return to the deprovisioning queue. | `2m` |
| deprovisioning.<br>workersAmount | Number of workers in deprovisioning queue. | `20` |
| catalog.<br>documentationUrl | Documentation URL used in the service catalog metadata | `https://help.sap.com/docs/btp/sap-business-technology-platform/provisioning-and-update-parameters-in-kyma-environment` |
| configPaths.catalog | Path to the service catalog configuration file. | `/config/catalog.yaml` |
| configPaths.<br>freemiumWhitelistedGlobalAccountIds | Path to the list of global account IDs that are allowed unlimited access to freemium (free) Kyma runtimes. Only accounts listed here can provision more than the default limit of free environments. | `/config/freemiumWhitelistedGlobalAccountIds.yaml` |
| configPaths.hapRule | Path to the rules for mapping plans and regions to hyperscaler account pools. | `/config/hapRule.yaml` |
| configPaths.<br>plansConfig | Path to the plans configuration file, which defines available service plans. | `/config/plansConfig.yaml` |
| configPaths.<br>providersConfig | Path to the providers configuration file, which defines hyperscaler/provider settings. | `/config/providersConfig.yaml` |
| configPaths.<br>quotaWhitelistedSubaccountIds | Path to the list of subaccount IDs that are allowed to bypass quota restrictions. | `/config/quotaWhitelistedSubaccountIds.yaml` |
| configPaths.<br>regionsSupportingMachine | Path to the list of regions that support machine-type selection. | `/config/regionsSupportingMachine.yaml` |
| configPaths.<br>skrDNSProvidersValues | Path to the DNS providers values. | `/config/skrDNSProvidersValues.yaml` |
| configPaths.<br>skrOIDCDefaultValues | Path to the default OIDC values. | `/config/skrOIDCDefaultValues.yaml` |
| configPaths.<br>trialRegionMapping | Path to the region mapping for trial environments. | `/config/trialRegionMapping.yaml` |
| configPaths.<br>cloudsqlSSLRootCert | Path to the Cloud SQL SSL root certificate file. | `/secrets/cloudsql-sslrootcert/server-ca.pem` |
| disableProcessOperationsInProgress | If true, the broker does NOT resume processing operations (provisioning, deprovisioning, updating, etc.) that were in progress when the broker process last stopped or restarted. | `false` |
| events.enabled | Enables or disables the events API and event storage for operation events (true/false). | `True` |
| freemiumWhitelistedGlobalAccountIds | List of global account IDs that are allowed unlimited access to freemium (free) Kyma runtimes. Only accounts listed here can provision more than the default limit of free environments. | `whitelist:` |
| gardener.<br>kubeconfigPath | Path to the kubeconfig file for accessing the Gardener cluster. | `/gardener/kubeconfig/kubeconfig` |
| gardener.project | Gardener project connected to SA for HAP credentials lookup. | `kyma-dev` |
| gardener.secretName | Name of the Kubernetes Secret containing Gardener credentials. | `gardener-credentials` |
| gardener.shootDomain | Default domain for shoots (clusters) created by Gardener. | `kyma-dev.shoot.canary.k8s-hana.ondemand.com` |
| hap.rule | Rules for mapping plans and regions to hyperscaler account pools. | `- aws  - aws(PR=cf-eu11) -> EU  - azure  - azure(PR=cf-ch20) -> EU  - gcp  - gcp(PR=cf-sa30) -> PR  - trial -> S  - sap-converged-cloud(HR=*) -> S  - azure_lite  - preview  - free` |
| infrastructureManager.<br>controlPlaneFailureTolerance | Sets the failure tolerance level for the Kubernetes control plane in Gardener clusters. Possible values: empty (default), "node", or "zone". | `` |
| infrastructureManager.<br>defaultShootPurpose | Sets the default purpose for Gardener shoots (clusters) created by the broker. Possible values: development, evaluation, production, testing. | `development` |
| infrastructureManager.<br>defaultTrialProvider | Sets the default cloud provider for trial Kyma runtimes, for example, Azure, AWS. | `Azure` |
| infrastructureManager.<br>ingressFilteringPlans | Comma-separated list of plan names for which ingress filtering is available. | `azure,gcp,aws` |
| infrastructureManager.<br>kubernetesVersion | Sets the default Kubernetes version for new clusters provisioned by the broker. | `1.16.9` |
| infrastructureManager.<br>machineImage | Sets the default machine image name for nodes in provisioned clusters. If empty, the Gardener default value is used. | `` |
| infrastructureManager.<br>machineImageVersion | Sets the version of the machine image for nodes in provisioned clusters. If empty, the Gardener default value is used. | `` |
| infrastructureManager.<br>multiZoneCluster | If true, enables provisioning of clusters with nodes distributed across multiple availability zones. | `false` |
| infrastructureManager.<br>useSmallerMachineTypes | If true, provisions trial, freemium, and azure_lite clusters using smaller machine types. | `false` |
| kubeconfig.<br>allowOrigins | Specifies which origins are allowed for Cross-Origin Resource Sharing (CORS) on the /kubeconfig endpoint. | `*` |
| kymaDashboardConfig.<br>landscapeURL | The base URL of the Kyma Dashboard used to generate links to the web UI for Kyma runtimes. | `https://dashboard.dev.kyma.cloud.sap` |
| metricsv2.enabled | If true, enables metricsv2 collection and Prometheus exposure. | `False` |
| metricsv2.<br>operationResultFinishedOperationRetentionPeriod | Duration of retaining finished operation results in memory. | `3h` |
| metricsv2.<br>operationResultPollingInterval | Frequency of polling for operation results. | `1m` |
| metricsv2.<br>operationResultRetentionPeriod | Duration of retaining operation results. | `1h` |
| metricsv2.<br>operationStatsPollingInterval | Frequency of polling for operation statistics. | `1m` |
| profiler.memory | Enables memory profiler (true/false). | `False` |
| quotaLimitCheck.<br>enabled | If true, validates during provisioning that the assigned quota for the subaccount is not exceeded. | `False` |
| quotaLimitCheck.<br>interval | The interval between requests to the Entitlements API in case of errors. | `1s` |
| quotaLimitCheck.<br>retries | The number of retry attempts made when the Entitlements API request fails. | `5` |
| quotaWhitelistedSubaccountIds | List of subaccount IDs that have unlimited quota for Kyma runtimes. Only subaccounts listed here can provision beyond their assigned quota limits. | `whitelist:` |
| regionsSupportingMachine | Defines which machine type families are available in which regions (and optionally, zones). Restricts provisioning of listed machine types to the specified regions/zones only. If a machine type is not listed, it is considered available in all regions. | `` |
| runtimeConfiguration | Defines the default KymaCR template. | `default: \|-      kyma-template: \|-        apiVersion: operator.kyma-project.io/v1beta2        kind: Kyma        metadata:          labels:            "operator.kyma-project.io/managed-by": "lifecycle-manager"          name: tbd          namespace: kcp-system        spec:          channel: fast          modules:            - name: api-gateway            - name: istio            - name: btp-operator      additional-components: []` |
| skrDNSProvidersValues | Contains DNS provider configuration for Kyma clusters. | `providers: []` |
| skrOIDCDefaultValues | Contains the default OIDC configuration for Kyma clusters. | `clientID: "9bd05ed7-a930-44e6-8c79-e6defeb7dec9"    groupsClaim: "groups"    groupsPrefix: "-"    issuerURL: "https://kymatest.accounts400.ondemand.com"    signingAlgs: [ "RS256" ]    usernameClaim: "sub"    usernamePrefix: "-"` |
| stepTimeouts.<br>checkRuntimeResourceCreate | Maximum time to wait for a runtime resource to be created before considering the step as failed. | `60m` |
| stepTimeouts.<br>checkRuntimeResourceDeletion | Maximum time to wait for a runtime resource to be deleted before considering the step as failed. | `60m` |
| stepTimeouts.<br>checkRuntimeResourceUpdate | Maximum time to wait for a runtime resource to be updated before considering the step as failed. | `180m` |
| testConfig.kebDeployment.<br>useAnnotations | - | `False` |
| testConfig.kebDeployment.<br>weight | - | `2` |
| trialRegionsMapping | Determines a Kyma region for a trial environment based on the requested platform region. | `cf-eu10: europe    cf-us10: us    cf-ap21: asia` |
| osbUpdateProcessingEnabled | If true, the broker processes update requests for service instances. | `true` |
| holdHAPSteps | If true, the broker holds any operation with HAP assignments. It is designed for migration (SecretBinding to CredentialBinding). | `false` |
| subscriptionGardenerResource | Name of the Gardener resource, which the broker uses to look up for hyperscaler assignment. Allowed values: SecretBinding or CredentialsBinding. | `SecretBinding` |
| machinesAvailabilityEndpoint | If true, the broker exposes the API endpoint that returns the availability of machine types. | `False` |
| clusterNameInKubeconfig | If true, the cluster name is used as the context name when generating the kubeconfig. | `False` |
| cis.accounts.authURL | The OAuth2 token endpoint (authorization URL) used to obtain access tokens for authenticating requests to the CIS Accounts API. | None |
| cis.accounts.id | The OAuth2 client ID used for authenticating requests to the CIS Accounts API. | None |
| cis.accounts.secret | The OAuth2 client secret used together with the client ID for authentication with the CIS Accounts API. | None |
| cis.accounts.<br>secretName | The name of the Kubernetes Secret containing the CIS Accounts client ID and secret. | `cis-creds-accounts` |
| cis.accounts.<br>serviceURL | The base URL of the CIS Accounts API endpoint, used for fetching subaccount data. | None |
| cis.accounts.<br>clientIdKey | The key in the Kubernetes Secret that contains the CIS v2 client ID. | `id` |
| cis.accounts.<br>secretKey | The key in the Kubernetes Secret that contains the CIS v2 client secret. | `secret` |
| cis.v2.authURL | The OAuth2 token endpoint (authorization URL) for CIS v2, used to obtain access tokens for authenticating requests. | None |
| cis.v2.<br>eventServiceURL | The endpoint URL for the CIS v2 event service, used to fetch subaccount events. | None |
| cis.v2.id | The OAuth2 client ID used for authenticating requests to the CIS v2 API. | None |
| cis.v2.secret | The OAuth2 client secret used together with the client ID for authentication with the CIS v2 API. | None |
| cis.v2.secretName | The name of the Kubernetes Secret containing the CIS v2 client ID and secret. | `cis-creds-v2` |
| cis.v2.jobRetries | The number of times a job should be retried in case of failure. | `6` |
| cis.v2.<br>maxRequestRetries | The maximum number of request retries to the CIS v2 API in case of errors. | `3` |
| cis.v2.<br>rateLimitingInterval | The minimum interval between requests to the CIS v2 API in case of errors. | `2s` |
| cis.v2.<br>requestInterval | The interval between requests to the CIS v2 API. | `200ms` |
| cis.v2.clientIdKey | The key in the Kubernetes Secret that contains the CIS v2 client ID. | `id` |
| cis.v2.secretKey | The key in the Kubernetes Secret that contains the CIS v2 client secret. | `secret` |
| cis.entitlements.<br>authURL | The OAuth2 token endpoint (authorization URL) used to obtain access tokens for authenticating requests to the CIS Entitlements API. | None |
| cis.entitlements.id | The OAuth2 client ID used for authenticating requests to the CIS Entitlements API. | None |
| cis.entitlements.<br>secret | The OAuth2 client secret used together with the client ID for authentication with the CIS Entitlements API. | None |
| cis.entitlements.<br>secretName | The name of the Kubernetes Secret containing the CIS Entitlements client ID and secret. | `cis-creds-entitlements` |
| cis.entitlements.<br>serviceURL | The base URL of the CIS Entitlements API endpoint, used for fetching quota assignments. | None |
| cis.entitlements.<br>clientIdKey | The key in the Kubernetes Secret that contains the CIS Entitlements client ID. | `id` |
| cis.entitlements.<br>secretKey | The key in the Kubernetes Secret that contains the CIS Entitlements client secret. | `secret` |
| deprovisionRetrigger.<br>dryRun | If true, the job runs in dry-run mode and does not actually retrigger deprovisioning. | `False` |
| deprovisionRetrigger.<br>enabled | If true, enables the Deprovision Retrigger CronJob, which periodically attempts to deprovision instances that were not fully deleted. | `True` |
| deprovisionRetrigger.<br>schedule | - | `0 2 * * *` |
| freeCleanup.dryRun | If true, the job only logs what would be deleted without actually removing any data. | `False` |
| freeCleanup.enabled | If true, enables the Free Cleanup CronJob. | `True` |
| freeCleanup.<br>expirationPeriod | Specifies how long a free instance can exist before being eligible for cleanup. | `2160h` |
| freeCleanup.planID | The ID of the free plan to be used for cleanup. | `b1a5764e-2ea1-4f95-94c0-2b4538b37b55` |
| freeCleanup.schedule | - | `0,15,30,45 * * * *` |
| freeCleanup.testRun | If true, runs the job in test mode (no real deletions, for testing purposes). | `False` |
| freeCleanup.<br>testSubaccountID | Subaccount ID used for test runs. | `prow-keb-trial-suspension` |
| globalaccounts.<br>dryRun | If true, runs the global accounts synchronization job in dry-run mode (no changes are made). | `False` |
| globalaccounts.<br>enabled | If true, enables the global accounts synchronization job. | `False` |
| globalaccounts.name | Name of the global accounts synchronization job or deployment. | `kyma-environment-globalaccounts` |
| migratorJobs.argosync.<br>enabled | If true, enables the ArgoCD sync job for schema migration. | `False` |
| migratorJobs.argosync.<br>syncwave | The sync wave value for ArgoCD hooks. | `0` |
| migratorJobs.<br>direction | Defines the direction of the schema migration, either "up" or "down". | `up` |
| migratorJobs.enabled | If true, enables all migrator jobs. | `True` |
| migratorJobs.helmhook.<br>enabled | If true, enables the Helm hook job for schema migration. | `True` |
| migratorJobs.helmhook.<br>weight | The weight value for the Helm hook. | `1` |
| oidc.groups.admin | - | `runtimeAdmin` |
| oidc.groups.operator | - | `runtimeOperator` |
| oidc.groups.<br>orchestrations | - | `orchestrationsAdmin` |
| oidc.groups.viewer | - | `runtimeViewer` |
| oidc.issuer | - | `https://kymatest.accounts400.ondemand.com` |
| oidc.issuers | - | `[]` |
| oidc.keysURL | - | `https://kymatest.accounts400.ondemand.com/oauth2/certs` |
| runtimeReconciler.<br>dryRun | If true, runs the reconciler in dry-run mode (no changes are made, only logs actions). | `False` |
| runtimeReconciler.<br>enabled | Enables or disables the Runtime Reconciler deployment. | `False` |
| runtimeReconciler.<br>jobEnabled | If true, enables the periodic reconciliation job. | `False` |
| runtimeReconciler.<br>jobInterval | Interval (in minutes) between reconciliation job runs. | `1440` |
| runtimeReconciler.<br>jobReconciliationDelay | Delay before starting reconciliation after job trigger. | `1s` |
| runtimeReconciler.<br>metricsPort | Port on which the reconciler exposes Prometheus metrics. | `8081` |
| serviceBindingCleanup.<br>dryRun | If true, the job only logs what would be deleted without actually removing any bindings. | `False` |
| serviceBindingCleanup.<br>enabled | If true, enables the Service Binding Cleanup CronJob. | `False` |
| serviceBindingCleanup.<br>requestRetries | Number of times to retry a failed DELETE request for a binding. | `2` |
| serviceBindingCleanup.<br>requestTimeout | Timeout for each DELETE request to the broker. | `2s` |
| serviceBindingCleanup.<br>schedule | - | `0 2,14 * * *` |
| subaccountCleanup.<br>enabled | - | `false` |
| subaccountCleanup.<br>nameV1 | - | `kcp-subaccount-cleaner-v1.0` |
| subaccountCleanup.<br>nameV2 | - | `kcp-subaccount-cleaner-v2.0` |
| subaccountCleanup.<br>schedule | - | `0 1 * * *` |
| subaccountCleanup.<br>clientV1VersionName | Client version. | `v1.0` |
| subaccountCleanup.<br>clientV2VersionName | Client version. | `v2.0` |
| subaccountSync.<br>accountSyncInterval | Interval between full account synchronization runs. | `24h` |
| subaccountSync.<br>alwaysSubaccountFromDatabase | If true, fetches subaccountID from the database only when the subaccount is empty. | `False` |
| subaccountSync.cisRateLimits.<br>accounts.<br>maxRequestsPerInterval | Maximum number of requests per interval to the CIS Accounts API. | `5` |
| subaccountSync.cisRateLimits.<br>accounts.<br>rateLimitingInterval | Minimum interval between requests to the CIS Accounts API. | `2s` |
| subaccountSync.cisRateLimits.<br>events.<br>maxRequestsPerInterval | Maximum number of requests per interval to the CIS Events API. | `5` |
| subaccountSync.cisRateLimits.<br>events.<br>rateLimitingInterval | Minimum interval between requests to the CIS Events API. | `2s` |
| subaccountSync.<br>enabled | If true, enables the subaccount synchronization job. | `True` |
| subaccountSync.<br>eventsWindowInterval | Time window for collecting events from CIS. | `15m` |
| subaccountSync.<br>eventsWindowSize | Size of the time window for collecting events from CIS. | `20m` |
| subaccountSync.<br>logLevel | Log level for the subaccount sync job. | `info` |
| subaccountSync.<br>metricsPort | Port on which the subaccount sync service exposes Prometheus metrics. | `8081` |
| subaccountSync.name | Name of the subaccount sync deployment. | `subaccount-sync` |
| subaccountSync.<br>queueSleepInterval | Interval between queue processing cycles. | `30s` |
| subaccountSync.<br>storageSyncInterval | Interval between storage synchronization. | `5m` |
| subaccountSync.<br>updateResources | If true, enables updating resources during subaccount sync. | `False` |
| trialCleanup.dryRun | If true, the job only logs what would be deleted without actually removing any data. | `False` |
| trialCleanup.enabled | If true, enables the Trial Cleanup CronJob, which removes expired trial Kyma runtimes. | `True` |
| trialCleanup.<br>expirationPeriod | Specifies how long a trial instance can exist before being expired. | `336h` |
| trialCleanup.planID | The ID of the trial plan to be used for cleanup. | `7d55d31d-35ae-4438-bf13-6ffdfa107d9f` |
| trialCleanup.<br>schedule | - | `15 1 * * *` |
| trialCleanup.testRun | If true, runs the job in test mode. | `False` |
| trialCleanup.<br>testSubaccountID | Subaccount ID used for test runs. | `prow-keb-trial-suspension` |
| serviceMonitor.<br>enabled | - | `False` |
| serviceMonitor.<br>interval | - | `30s` |
| serviceMonitor.<br>scrapeTimeout | - | `10s` |
| vmscrapes.enabled | - | `True` |
| vmscrapes.interval | - | `30s` |
| vmscrapes.<br>scrapeTimeout | - | `10s` |
| vsoSecrets.secrets.cis-v2.<br>path | - | `cis` |
| vsoSecrets.secrets.cis-v2.<br>secretName | - | `{{ .Values.cis.v2.secretName \| required "please specify .Values.cis.v2.secretName"}}` |
| vsoSecrets.secrets.cis-v2.<br>restartTargets | - | `- {'kind': 'Deployment', 'name': '{{- .Values.subaccountSync.name -}}'}` |
| vsoSecrets.secrets.cis-v2.<br>labels | - | `{{ template "kyma-env-broker.labels" . }}` |
| vsoSecrets.secrets.cis-v2.<br>templating.enabled | - | `True` |
| vsoSecrets.secrets.cis-v2.<br>templating.keys.id | - | `v2_id` |
| vsoSecrets.secrets.cis-v2.<br>templating.keys.<br>secret | - | `v2_secret` |
| vsoSecrets.secrets.cis-accounts.<br>path | - | `cis` |
| vsoSecrets.secrets.cis-accounts.<br>secretName | - | `{{ .Values.cis.accounts.secretName \| required "please specify .Values.cis.accounts.secretName"}}` |
| vsoSecrets.secrets.cis-accounts.<br>restartTargets | - | `- {'kind': 'Deployment', 'name': '{{- .Values.subaccountSync.name -}}'}` |
| vsoSecrets.secrets.cis-accounts.<br>labels | - | `{{ template "kyma-env-broker.labels" . }}` |
| vsoSecrets.secrets.cis-accounts.<br>templating.enabled | - | `True` |
| vsoSecrets.secrets.cis-accounts.<br>templating.keys.id | - | `account_id` |
| vsoSecrets.secrets.cis-accounts.<br>templating.keys.<br>secret | - | `account_secret` |
| vsoSecrets.secrets.cis-entitlements.<br>path | - | `cis` |
| vsoSecrets.secrets.cis-entitlements.<br>secretName | - | `{{ .Values.cis.entitlements.secretName \| required "please specify .Values.cis.entitlements.secretName"}}` |
| vsoSecrets.secrets.cis-entitlements.<br>restartTargets | - | `- {'kind': 'Deployment', 'name': '{{- template "kyma-env-broker.fullname" . -}}'}` |
| vsoSecrets.secrets.cis-entitlements.<br>labels | - | `{{ template "kyma-env-broker.labels" . }}` |
| vsoSecrets.secrets.cis-entitlements.<br>templating.enabled | - | `True` |
| vsoSecrets.secrets.cis-entitlements.<br>templating.keys.id | - | `entitlements_id` |
| vsoSecrets.secrets.cis-entitlements.<br>templating.keys.<br>secret | - | `entitlements_secret` |

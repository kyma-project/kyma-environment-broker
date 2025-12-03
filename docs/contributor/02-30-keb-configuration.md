## Kyma Environment Broker Configuration

Kyma Environment Broker (KEB) binary allows you to override some configuration parameters. You can specify the following environment variables:

| Environment Variable | Current Value | Description |
|---------------------|------------------------------|---------------------------------------------------------------|
| **APP_BROKER_BINDING_&#x200b;BINDABLE_PLANS** | <code>aws</code> | Comma-separated list of plan names for which service binding is enabled, for example, "aws,gcp". |
| **APP_BROKER_BINDING_&#x200b;CREATE_BINDING_&#x200b;TIMEOUT** | <code>15s</code> | Timeout for creating a binding, for example, 15s, 1m. |
| **APP_BROKER_BINDING_&#x200b;ENABLED** | <code>false</code> | Enables or disables the service binding endpoint (true/false). |
| **APP_BROKER_BINDING_&#x200b;EXPIRATION_SECONDS** | <code>600</code> | Default expiration time (in seconds) for a binding if not specified in the request. |
| **APP_BROKER_BINDING_&#x200b;MAX_BINDINGS_COUNT** | <code>10</code> | Maximum number of non-expired bindings allowed per instance. |
| **APP_BROKER_BINDING_&#x200b;MAX_EXPIRATION_&#x200b;SECONDS** | <code>7200</code> | Maximum allowed expiration time (in seconds) for a binding. |
| **APP_BROKER_BINDING_&#x200b;MIN_EXPIRATION_&#x200b;SECONDS** | <code>600</code> | Minimum allowed expiration time (in seconds) for a binding. Can't be lower than 600 seconds. Forced by Gardener. |
| **APP_BROKER_CHECK_&#x200b;QUOTA_LIMIT** | <code>false</code> | If true, validates during provisioning that the assigned quota for the subaccount is not exceeded. |
| **APP_BROKER_DEFAULT_&#x200b;REQUEST_REGION** | <code>cf-eu10</code> | Default platform region for requests if not specified. |
| **APP_BROKER_DUAL_&#x200b;STACK_DOCS_URL** | <code>https://help.sap.com/docs/btp/sap-business-technology-platform/kyma-runtime-with-dual-stack-support</code> | URL to the documentation for dual-stack networking. Used in dual-stack configuration description in schema. |
| **APP_BROKER_ENABLE_&#x200b;PLANS** | <code>azure,gcp,azure_lite,trial,aws</code> | Comma-separated list of plan names enabled and available for provisioning in KEB. |
| **APP_BROKER_ENABLE_&#x200b;PLAN_UPGRADES** | <code>false</code> | If true, allows users to upgrade their plans (if a plan supports upgrades). |
| **APP_BROKER_FREE_&#x200b;DOCS_URL** | <code>https://help.sap.com/docs/btp/sap-business-technology-platform/using-free-service-plans?version=Cloud</code> | URL to the documentation of free Kyma runtimes. Used in API responses and UI labels to direct users to help or documentation about free plans |
| **APP_BROKER_FREE_&#x200b;EXPIRATION_PERIOD** | <code>720h</code> | Determines when to show expiration info to users. |
| **APP_BROKER_GARDENER_&#x200b;SEEDS_CACHE_CONFIG_&#x200b;MAP_NAME** | <code>gardener-seeds-cache</code> | Name of the Kubernetes ConfigMap used as a cache for Gardener seeds. |
| **APP_BROKER_MONITOR_&#x200b;ADDITIONAL_&#x200b;PROPERTIES** | <code>false</code> | If true, collects properties from the provisioning request that are not explicitly defined in the schema and stores them in persistent storage. |
| **APP_BROKER_ONLY_ONE_&#x200b;FREE_PER_GA** | <code>false</code> | If true, restricts each global account to only one freemium (free) Kyma runtime. When enabled, provisioning another free environment for the same global account is blocked even if the previous one is deprovisioned. |
| **APP_BROKER_ONLY_&#x200b;SINGLE_TRIAL_PER_GA** | <code>true</code> | If true, restricts each global account to only one active trial Kyma runtime at a time. When enabled, provisioning another trial environment for the same global account is blocked until the previous one is deprovisioned. |
| **APP_BROKER_&#x200b;OPERATION_TIMEOUT** | <code>7h</code> | Maximum allowed duration for processing a single operation (provisioning, deprovisioning, etc.). If the operation exceeds this timeout, it is marked as failed. |
| **APP_BROKER_PORT** | <code>8080</code> | Port for the broker HTTP server. |
| **APP_BROKER_REJECT_&#x200b;UNSUPPORTED_&#x200b;PARAMETERS** | <code>false</code> | If true, rejects requests that contain parameters that are not defined in schemas. |
| **APP_BROKER_STATUS_&#x200b;PORT** | <code>8071</code> | Port for the broker status/health endpoint. |
| **APP_BROKER_&#x200b;SUBACCOUNT_MOVEMENT_&#x200b;ENABLED** | <code>false</code> | If true, enables subaccount movement (allows changing global account for an instance). |
| **APP_BROKER_TRIAL_&#x200b;DOCS_URL** | <code>https://help.sap.com/docs/</code> | URL to the documentation for trial Kyma runtimes. Used in API responses and UI labels. |
| **APP_BROKER_UPDATE_&#x200b;CUSTOM_RESOURCES_&#x200b;LABELS_ON_ACCOUNT_&#x200b;MOVE** | <code>false</code> | If true, updates runtimeCR labels when moving subaccounts. |
| **APP_BROKER_URL** | <code>kyma-env-broker.localhost</code> | - |
| **APP_CATALOG_FILE_&#x200b;PATH** | <code>/config/catalog.yaml</code> | Path to the service catalog configuration file. |
| **APP_DATABASE_FIPS_&#x200b;REWRITE_CFB** | <code>false</code> | Rewrite old data using AES-GCM encryption. |
| **APP_DATABASE_FIPS_&#x200b;WRITE_GCM** | <code>false</code> | Write using AES-GCM encryption. |
| **APP_DATABASE_HOST** | None | Specifies the host of the database. |
| **APP_DATABASE_NAME** | None | Specifies the name of the database. |
| **APP_DATABASE_&#x200b;PASSWORD** | None | Specifies the user password for the database. |
| **APP_DATABASE_PORT** | None | Specifies the port for the database. |
| **APP_DATABASE_SECRET_&#x200b;KEY** | None | Specifies the Secret key for the database. |
| **APP_DATABASE_SSLMODE** | None | Activates the SSL mode for PostgreSQL. |
| **APP_DATABASE_&#x200b;SSLROOTCERT** | <code>/secrets/cloudsql-sslrootcert/server-ca.pem</code> | Path to the Cloud SQL SSL root certificate file. |
| **APP_DATABASE_USER** | None | Specifies the username for the database. |
| **APP_DEPROVISIONING_&#x200b;MAX_STEP_PROCESSING_&#x200b;TIME** | <code>2m</code> | Maximum time a worker is allowed to process a step before it must return to the deprovisioning queue. |
| **APP_DEPROVISIONING_&#x200b;WORKERS_AMOUNT** | <code>20</code> | Number of workers in deprovisioning queue. |
| **APP_DISABLE_PROCESS_&#x200b;OPERATIONS_IN_&#x200b;PROGRESS** | <code>false</code> | If true, the broker does NOT resume processing operations (provisioning, deprovisioning, updating, etc.) that were in progress when the broker process last stopped or restarted. |
| **APP_DOMAIN_NAME** | <code>localhost</code> | - |
| **APP_EVENTS_ENABLED** | <code>true</code> | Enables or disables the events API and event storage for operation events (true/false). |
| **APP_FREEMIUM_&#x200b;WHITELISTED_GLOBAL_&#x200b;ACCOUNTS_FILE_PATH** | <code>/config/freemiumWhitelistedGlobalAccountIds.yaml</code> | Path to the list of global account IDs that are allowed unlimited access to freemium (free) Kyma runtimes. Only accounts listed here can provision more than the default limit of free environments. |
| **APP_GARDENER_&#x200b;KUBECONFIG_PATH** | <code>/gardener/kubeconfig/kubeconfig</code> | Path to the kubeconfig file for accessing the Gardener cluster. |
| **APP_GARDENER_PROJECT** | <code>kyma-dev</code> | Gardener project connected to SA for HAP credentials lookup. |
| **APP_GARDENER_SHOOT_&#x200b;DOMAIN** | <code>kyma-dev.shoot.canary.k8s-hana.ondemand.com</code> | Default domain for shoots (clusters) created by Gardener. |
| **APP_HAP_RULE_FILE_&#x200b;PATH** | <code>/config/hapRule.yaml</code> | Path to the rules for mapping plans and regions to hyperscaler account pools. |
| **APP_HOLD_HAP_STEPS** | <code>false</code> | If true, the broker holds any operation with HAP assignments. It is designed for migration (SecretBinding to CredentialBinding). |
| **APP_INFRASTRUCTURE_&#x200b;MANAGER_CONTROL_&#x200b;PLANE_FAILURE_&#x200b;TOLERANCE** | None | Sets the failure tolerance level for the Kubernetes control plane in Gardener clusters. Possible values: empty (default), "node", or "zone". |
| **APP_INFRASTRUCTURE_&#x200b;MANAGER_DEFAULT_&#x200b;GARDENER_SHOOT_&#x200b;PURPOSE** | <code>development</code> | Sets the default purpose for Gardener shoots (clusters) created by the broker. Possible values: development, evaluation, production, testing. |
| **APP_INFRASTRUCTURE_&#x200b;MANAGER_DEFAULT_&#x200b;TRIAL_PROVIDER** | <code>Azure</code> | Sets the default cloud provider for trial Kyma runtimes, for example, Azure, AWS. |
| **APP_INFRASTRUCTURE_&#x200b;MANAGER_INGRESS_&#x200b;FILTERING_PLANS** | <code>azure,gcp,aws</code> | Comma-separated list of plan names for which ingress filtering is available. |
| **APP_INFRASTRUCTURE_&#x200b;MANAGER_KUBERNETES_&#x200b;VERSION** | <code>1.16.9</code> | Sets the default Kubernetes version for new clusters provisioned by the broker. |
| **APP_INFRASTRUCTURE_&#x200b;MANAGER_MACHINE_&#x200b;IMAGE** | None | Sets the default machine image name for nodes in provisioned clusters. If empty, the Gardener default value is used. |
| **APP_INFRASTRUCTURE_&#x200b;MANAGER_MACHINE_&#x200b;IMAGE_VERSION** | None | Sets the version of the machine image for nodes in provisioned clusters. If empty, the Gardener default value is used. |
| **APP_INFRASTRUCTURE_&#x200b;MANAGER_MULTI_ZONE_&#x200b;CLUSTER** | <code>false</code> | If true, enables provisioning of clusters with nodes distributed across multiple availability zones. |
| **APP_INFRASTRUCTURE_&#x200b;MANAGER_USE_SMALLER_&#x200b;MACHINE_TYPES** | <code>false</code> | If true, provisions trial, freemium, and azure_lite clusters using smaller machine types. |
| **APP_KUBECONFIG_&#x200b;ALLOW_ORIGINS** | <code>*</code> | Specifies which origins are allowed for Cross-Origin Resource Sharing (CORS) on the /kubeconfig endpoint. |
| **APP_KYMA_DASHBOARD_&#x200b;CONFIG_LANDSCAPE_URL** | <code>https://dashboard.dev.kyma.cloud.sap</code> | The base URL of the Kyma Dashboard used to generate links to the web UI for Kyma runtimes. |
| **APP_MACHINES_&#x200b;AVAILABILITY_&#x200b;ENDPOINT** | <code>false</code> | If true, the broker exposes the API endpoint that returns the availability of machine types. |
| **APP_METRICSV2_&#x200b;ENABLED** | <code>false</code> | If true, enables metricsv2 collection and Prometheus exposure. |
| **APP_METRICSV2_&#x200b;OPERATION_RESULT_&#x200b;FINISHED_OPERATION_&#x200b;RETENTION_PERIOD** | <code>3h</code> | Duration of retaining finished operation results in memory. |
| **APP_METRICSV2_&#x200b;OPERATION_RESULT_&#x200b;POLLING_INTERVAL** | <code>1m</code> | Frequency of polling for operation results. |
| **APP_METRICSV2_&#x200b;OPERATION_RESULT_&#x200b;RETENTION_PERIOD** | <code>1h</code> | Duration of retaining operation results. |
| **APP_METRICSV2_&#x200b;OPERATION_STATS_&#x200b;POLLING_INTERVAL** | <code>1m</code> | Frequency of polling for operation statistics. |
| **APP_PLANS_&#x200b;CONFIGURATION_FILE_&#x200b;PATH** | <code>/config/plansConfig.yaml</code> | Path to the plans configuration file, which defines available service plans. |
| **APP_PROFILER_MEMORY** | <code>false</code> | Enables memory profiler (true/false). |
| **APP_PROVIDERS_&#x200b;CONFIGURATION_FILE_&#x200b;PATH** | <code>/config/providersConfig.yaml</code> | Path to the providers configuration file, which defines hyperscaler/provider settings. |
| **APP_PROVISIONING_&#x200b;MAX_STEP_PROCESSING_&#x200b;TIME** | <code>2m</code> | Maximum time a worker is allowed to process a step before it must return to the provisioning queue. |
| **APP_PROVISIONING_&#x200b;WORKERS_AMOUNT** | <code>20</code> | Number of workers in provisioning queue. |
| **APP_QUOTA_AUTH_URL** | <code>TBD</code> | The OAuth2 token endpoint (authorization URL) used to obtain access tokens for authenticating requests to the CIS Entitlements API. |
| **APP_QUOTA_CLIENT_ID** | None | Specifies the client ID for the OAuth2 authentication in CIS Entitlements API. |
| **APP_QUOTA_CLIENT_&#x200b;SECRET** | None | Specifies the client secret for the OAuth2 authentication in CIS Entitlements API. |
| **APP_QUOTA_INTERVAL** | <code>1s</code> | The interval between requests to the Entitlements API in case of errors. |
| **APP_QUOTA_RETRIES** | <code>5</code> | The number of retry attempts made when the Entitlements API request fails. |
| **APP_QUOTA_SERVICE_&#x200b;URL** | <code>TBD</code> | The base URL of the CIS Entitlements API endpoint, used for fetching quota assignments. |
| **APP_QUOTA_&#x200b;WHITELISTED_&#x200b;SUBACCOUNTS_FILE_&#x200b;PATH** | <code>/config/quotaWhitelistedSubaccountIds.yaml</code> | Path to the list of subaccount IDs that are allowed to bypass quota restrictions. |
| **APP_REGIONS_&#x200b;SUPPORTING_MACHINE_&#x200b;FILE_PATH** | <code>/config/regionsSupportingMachine.yaml</code> | Path to the list of regions that support machine-type selection. |
| **APP_RUNTIME_&#x200b;CONFIGURATION_&#x200b;CONFIG_MAP_NAME** | None | Name of the ConfigMap with the default KymaCR template. |
| **APP_SKR_DNS_&#x200b;PROVIDERS_VALUES_&#x200b;YAML_FILE_PATH** | <code>/config/skrDNSProvidersValues.yaml</code> | Path to the DNS providers values. |
| **APP_SKR_OIDC_&#x200b;DEFAULT_VALUES_YAML_&#x200b;FILE_PATH** | <code>/config/skrOIDCDefaultValues.yaml</code> | Path to the default OIDC values. |
| **APP_STEP_TIMEOUTS_&#x200b;CHECK_RUNTIME_&#x200b;RESOURCE_CREATE** | <code>60m</code> | Maximum time to wait for a runtime resource to be created before considering the step as failed. |
| **APP_STEP_TIMEOUTS_&#x200b;CHECK_RUNTIME_&#x200b;RESOURCE_DELETION** | <code>60m</code> | Maximum time to wait for a runtime resource to be deleted before considering the step as failed. |
| **APP_STEP_TIMEOUTS_&#x200b;CHECK_RUNTIME_&#x200b;RESOURCE_UPDATE** | <code>180m</code> | Maximum time to wait for a runtime resource to be updated before considering the step as failed. |
| **APP_SUBSCRIPTION_&#x200b;GARDENER_RESOURCE** | <code>SecretBinding</code> | Name of the Gardener resource, which the broker uses to look up for hyperscaler assignment. Allowed values: SecretBinding or CredentialsBinding. |
| **APP_TRIAL_REGION_&#x200b;MAPPING_FILE_PATH** | <code>/config/trialRegionMapping.yaml</code> | Path to the region mapping for trial environments. |
| **APP_UPDATE_MAX_STEP_&#x200b;PROCESSING_TIME** | <code>2m</code> | Maximum time a worker is allowed to process a step before it must return to the update queue. |
| **APP_UPDATE_&#x200b;PROCESSING_ENABLED** | <code>true</code> | If true, the broker processes update requests for service instances. |
| **APP_UPDATE_WORKERS_&#x200b;AMOUNT** | <code>20</code> | Number of workers in update queue. |
| **APP_USE_HAP_FOR_&#x200b;DEPROVISIONING** | <code>false</code> | If true, uses HAP for deprovisioning. |

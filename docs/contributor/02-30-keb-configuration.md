## Kyma Environment Broker Configuration

Kyma Environment Broker (KEB) binary allows you to override some configuration parameters. You can specify the following environment variables:

| Environment Variable | Current Value | Description |
|---------------------|------------------------------|---------------------------------------------------------------|
| **APP_ARCHIVING_DRY_&#x200b;RUN** | <code>true</code> | If true, runs the archiving process in dry-run mode: Makes no changes to the database, only logs what is to be archived or deleted |
| **APP_ARCHIVING_&#x200b;ENABLED** | <code>false</code> | If true, enables the archiving mechanism, which stores data about deprovisioned instances in an archive table at the end of the deprovisioning process |
| **APP_BROKER_ALLOW_&#x200b;UPDATE_EXPIRED_&#x200b;INSTANCE_WITH_&#x200b;CONTEXT** | <code>false</code> | Allow update of expired instance |
| **APP_BROKER_BINDING_&#x200b;BINDABLE_PLANS** | <code>aws</code> | Comma-separated list of plan names for which service binding is enabled, for example, "aws,gcp" |
| **APP_BROKER_BINDING_&#x200b;CREATE_BINDING_&#x200b;TIMEOUT** | <code>15s</code> | Timeout for creating a binding, for example, 15s, 1m |
| **APP_BROKER_BINDING_&#x200b;ENABLED** | <code>false</code> | Enables or disables the service binding endpoint (true/false) |
| **APP_BROKER_BINDING_&#x200b;EXPIRATION_SECONDS** | <code>600</code> | Default expiration time (in seconds) for a binding if not specified in the request |
| **APP_BROKER_BINDING_&#x200b;MAX_BINDINGS_COUNT** | <code>10</code> | Maximum number of non-expired bindings allowed per instance |
| **APP_BROKER_BINDING_&#x200b;MAX_EXPIRATION_&#x200b;SECONDS** | <code>7200</code> | Maximum allowed expiration time (in seconds) for a binding |
| **APP_BROKER_BINDING_&#x200b;MIN_EXPIRATION_&#x200b;SECONDS** | <code>600</code> | Minimum allowed expiration time (in seconds) for a binding. Can't be lower than 600 seconds. Forced by Gardener |
| **APP_BROKER_DEFAULT_&#x200b;REQUEST_REGION** | <code>cf-eu10</code> | Default platform region for requests if not specified |
| **APP_BROKER_DISABLE_&#x200b;SAP_CONVERGED_CLOUD** | <code>false</code> | If true, disables the SAP Cloud Infrastructure plan in the KEB. When set to true, users cannot provision SAP Cloud Infrastructure clusters |
| **APP_BROKER_ENABLE_&#x200b;PLANS** | <code>azure,gcp,azure_lite,trial,aws</code> | Comma-separated list of plan names enabled and available for provisioning in KEB |
| **APP_BROKER_ENABLE_&#x200b;PLAN_UPGRADES** | <code>false</code> | If true, allows users to upgrade their plans (if a plan supports upgrades) |
| **APP_BROKER_ENABLE_&#x200b;SHOOT_AND_SEED_SAME_&#x200b;REGION** | <code>false</code> | If true, enforces that the Gardener seed is placed in the same region as the shoot region selected during provisioning |
| **APP_BROKER_FREE_&#x200b;DOCS_URL** | <code>https://help.sap.com/docs/</code> | URL to the documentation of free Kyma runtimes. Used in API responses and UI labels to direct users to help or documentation about free plans |
| **APP_BROKER_FREE_&#x200b;EXPIRATION_PERIOD** | <code>720h</code> | Determines when to show expiration info to users |
| **APP_BROKER_GARDENER_&#x200b;SEEDS_CACHE_CONFIG_&#x200b;MAP_NAME** | None | - |
| **APP_BROKER_INCLUDE_&#x200b;ADDITIONAL_PARAMS_&#x200b;IN_SCHEMA** | <code>false</code> | If true, additional (advanced or less common) parameters are included in the provisioning schema for service plans |
| **APP_BROKER_MONITOR_&#x200b;ADDITIONAL_&#x200b;PROPERTIES** | <code>false</code> | If true, collects properties from the provisioning request that are not explicitly defined in the schema and stores them in persistent storage |
| **APP_BROKER_ONLY_ONE_&#x200b;FREE_PER_GA** | <code>false</code> | If true, restricts each global account to only one free (freemium) Kyma runtime. When enabled, provisioning another free environment for the same global account is blocked even if the previous one is deprovisioned |
| **APP_BROKER_ONLY_&#x200b;SINGLE_TRIAL_PER_GA** | <code>true</code> | If true, restricts each global account to only one active trial Kyma runtime at a time When enabled, provisioning another trial environment for the same global account is blocked until the previous one is deprovisioned |
| **APP_BROKER_&#x200b;OPERATION_TIMEOUT** | <code>7h</code> | Maximum allowed duration for processing a single operation (provisioning, deprovisioning, etc.) If the operation exceeds this timeout, it is marked as failed. Example: "7h" for 7 hours |
| **APP_BROKER_PORT** | <code>8080</code> | Port for the broker HTTP server |
| **APP_BROKER_REJECT_&#x200b;UNSUPPORTED_&#x200b;PARAMETERS** | <code>false</code> | If true, rejects requests that contain parameters that are not defined in schemas |
| **APP_BROKER_SHOW_&#x200b;FREE_EXPIRATION_INFO** | <code>false</code> | If true, adds expiration information for free plan Kyma runtimes to API responses and UI labels |
| **APP_BROKER_SHOW_&#x200b;TRIAL_EXPIRATION_&#x200b;INFO** | <code>false</code> | If true, adds expiration information for trial plan Kyma runtimes to API responses and UI labels |
| **APP_BROKER_STATUS_&#x200b;PORT** | <code>8071</code> | Port for the broker status/health endpoint |
| **APP_BROKER_&#x200b;SUBACCOUNT_MOVEMENT_&#x200b;ENABLED** | <code>false</code> | If true, enables subaccount movement (allows changing global account for an instance) |
| **APP_BROKER_&#x200b;SUBACCOUNTS_IDS_TO_&#x200b;SHOW_TRIAL_&#x200b;EXPIRATION_INFO** | <code>a45be5d8-eddc-4001-91cf-48cc644d571f</code> | Shows trial expiration information for specific subaccounts in the UI and API responses |
| **APP_BROKER_TRIAL_&#x200b;DOCS_URL** | <code>https://help.sap.com/docs/</code> | URL to the documentation for trial Kyma runtimes. Used in API responses and UI labels |
| **APP_BROKER_UPDATE_&#x200b;CUSTOM_RESOURCES_&#x200b;LABELS_ON_ACCOUNT_&#x200b;MOVE** | <code>false</code> | If true, updates runtimeCR labels when moving subaccounts |
| **APP_BROKER_URL** | <code>kyma-env-broker.localhost</code> | - |
| **APP_BROKER_USE_&#x200b;ADDITIONAL_OIDC_&#x200b;SCHEMA** | <code>false</code> | If true, enables the new list-based OIDC schema, allowing multiple OIDC configurations for a runtime |
| **APP_CATALOG_FILE_&#x200b;PATH** | None | - |
| **APP_CLEANING_DRY_RUN** | <code>true</code> | If true, the cleaning process runs in dry-run mode and does not actually delete any data from the database |
| **APP_CLEANING_ENABLED** | <code>false</code> | If true, enables the cleaning process, which removes all data about deprovisioned instances from the database |
| **APP_DATABASE_HOST** | None | - |
| **APP_DATABASE_NAME** | None | - |
| **APP_DATABASE_&#x200b;PASSWORD** | None | - |
| **APP_DATABASE_PORT** | None | - |
| **APP_DATABASE_SECRET_&#x200b;KEY** | None | - |
| **APP_DATABASE_SSLMODE** | None | - |
| **APP_DATABASE_&#x200b;SSLROOTCERT** | None | - |
| **APP_DATABASE_USER** | None | - |
| **APP_DEPROVISIONING_&#x200b;MAX_STEP_PROCESSING_&#x200b;TIME** | <code>2m</code> | Maximum time a worker is allowed to process a step before it must return to the deprovisioning queue |
| **APP_DEPROVISIONING_&#x200b;WORKERS_AMOUNT** | <code>20</code> | Number of workers in deprovisioning queue |
| **APP_DISABLE_PROCESS_&#x200b;OPERATIONS_IN_&#x200b;PROGRESS** | <code>false</code> | If true, the broker does NOT resume processing operations (provisioning, deprovisioning, updating, etc.) that were in progress when the broker process last stopped or restarted |
| **APP_DOMAIN_NAME** | <code>localhost</code> | - |
| **APP_EDP_ADMIN_URL** | <code>TBD</code> | Base URL for the EDP admin API |
| **APP_EDP_AUTH_URL** | <code>TBD</code> | OAuth2 token endpoint for EDP |
| **APP_EDP_DISABLED** | <code>true</code> | If true, disables EDP integration |
| **APP_EDP_ENVIRONMENT** | <code>dev</code> | EDP environment, for example, dev, prod |
| **APP_EDP_NAMESPACE** | <code>kyma-dev</code> | EDP namespace to use |
| **APP_EDP_REQUIRED** | <code>false</code> | If true, EDP integration is required for provisioning |
| **APP_EDP_SECRET** | None | - |
| **APP_EVENTS_ENABLED** | <code>true</code> | Enables or disables the /events API and event storage for operation events (true/false) |
| **APP_FREEMIUM_&#x200b;WHITELISTED_GLOBAL_&#x200b;ACCOUNTS_FILE_PATH** | None | - |
| **APP_GARDENER_&#x200b;KUBECONFIG_PATH** | <code>/gardener/kubeconfig/kubeconfig</code> | Path to the kubeconfig file for accessing the Gardener cluster |
| **APP_GARDENER_PROJECT** | <code>kyma-dev</code> | Gardener project connected to SA for HAP credentials lookup |
| **APP_GARDENER_SHOOT_&#x200b;DOMAIN** | <code>kyma-dev.shoot.canary.k8s-hana.ondemand.com</code> | Default domain for shoots (clusters) created by Gardener |
| **APP_HAP_RULE_FILE_&#x200b;PATH** | None | - |
| **APP_INFRASTRUCTURE_&#x200b;MANAGER_CONTROL_&#x200b;PLANE_FAILURE_&#x200b;TOLERANCE** | None | Sets the failure tolerance level for the Kubernetes control plane in Gardener clusters Possible values: empty (default), "node", or "zone" |
| **APP_INFRASTRUCTURE_&#x200b;MANAGER_DEFAULT_&#x200b;GARDENER_SHOOT_&#x200b;PURPOSE** | <code>development</code> | Sets the default purpose for Gardener shoots (clusters) created by the broker Possible values: development, evaluation, production, testing |
| **APP_INFRASTRUCTURE_&#x200b;MANAGER_DEFAULT_&#x200b;TRIAL_PROVIDER** | <code>Azure</code> | Sets the default cloud provider for trial Kyma runtimes, for example, Azure, AWS |
| **APP_INFRASTRUCTURE_&#x200b;MANAGER_INGRESS_&#x200b;FILTERING_PLANS** | <code>azure,gcp,aws</code> | Comma-separated list of plan names for which ingress filtering is available |
| **APP_INFRASTRUCTURE_&#x200b;MANAGER_KUBERNETES_&#x200b;VERSION** | <code>1.16.9</code> | Sets the default Kubernetes version for new clusters provisioned by the broker |
| **APP_INFRASTRUCTURE_&#x200b;MANAGER_MACHINE_&#x200b;IMAGE** | None | Sets the default machine image name for nodes in provisioned clusters. If empty, the Gardener default value is used |
| **APP_INFRASTRUCTURE_&#x200b;MANAGER_MACHINE_&#x200b;IMAGE_VERSION** | None | Sets the version of the machine image for nodes in provisioned clusters. If empty, the Gardener default value is used |
| **APP_INFRASTRUCTURE_&#x200b;MANAGER_MULTI_ZONE_&#x200b;CLUSTER** | <code>false</code> | If true, enables provisioning of clusters with nodes distributed across multiple availability zones |
| **APP_INFRASTRUCTURE_&#x200b;MANAGER_USE_SMALLER_&#x200b;MACHINE_TYPES** | <code>false</code> | If true, provisions trial, freemium, and azure_lite clusters using smaller machine types |
| **APP_KUBECONFIG_&#x200b;ALLOW_ORIGINS** | <code>*</code> | Specifies which origins are allowed for Cross-Origin Resource Sharing (CORS) on the /kubeconfig endpoint |
| **APP_KYMA_DASHBOARD_&#x200b;CONFIG_LANDSCAPE_URL** | <code>https://dashboard.dev.kyma.cloud.sap</code> | The base URL of the Kyma Dashboard used to generate links to the web UI for Kyma runtimes |
| **APP_LIFECYCLE_&#x200b;MANAGER_INTEGRATION_&#x200b;DISABLED** | <code>false</code> | When disabled, the broker does not create, update, or delete the KymaCR |
| **APP_METRICSV2_&#x200b;ENABLED** | <code>false</code> | If true, enables metricsv2 collection and Prometheus exposure |
| **APP_METRICSV2_&#x200b;OPERATION_RESULT_&#x200b;FINISHED_OPERATION_&#x200b;RETENTION_PERIOD** | <code>3h</code> | Duration of retaining finished operation results in memory |
| **APP_METRICSV2_&#x200b;OPERATION_RESULT_&#x200b;POLLING_INTERVAL** | <code>1m</code> | Frequency of polling for operation results |
| **APP_METRICSV2_&#x200b;OPERATION_RESULT_&#x200b;RETENTION_PERIOD** | <code>1h</code> | Duration of retaining operation results |
| **APP_METRICSV2_&#x200b;OPERATION_STATS_&#x200b;POLLING_INTERVAL** | <code>1m</code> | Frequency of polling for operation statistics |
| **APP_MULTIPLE_&#x200b;CONTEXTS** | <code>false</code> | If true, generates kubeconfig files with multiple contexts (if possible) instead of a single context |
| **APP_PLANS_&#x200b;CONFIGURATION_FILE_&#x200b;PATH** | None | - |
| **APP_PROFILER_MEMORY** | <code>false</code> | Enables memory profiler (true/false) |
| **APP_PROVIDERS_&#x200b;CONFIGURATION_FILE_&#x200b;PATH** | None | - |
| **APP_PROVISIONING_&#x200b;MAX_STEP_PROCESSING_&#x200b;TIME** | <code>2m</code> | Maximum time a worker is allowed to process a step before it must return to the provisioning queue |
| **APP_PROVISIONING_&#x200b;WORKERS_AMOUNT** | <code>20</code> | Number of workers in provisioning queue |
| **APP_REGIONS_&#x200b;SUPPORTING_MACHINE_&#x200b;FILE_PATH** | None | - |
| **APP_RUNTIME_&#x200b;CONFIGURATION_&#x200b;CONFIG_MAP_NAME** | None | - |
| **APP_SKR_DNS_&#x200b;PROVIDERS_VALUES_&#x200b;YAML_FILE_PATH** | None | - |
| **APP_SKR_OIDC_&#x200b;DEFAULT_VALUES_YAML_&#x200b;FILE_PATH** | None | - |
| **APP_STEP_TIMEOUTS_&#x200b;CHECK_RUNTIME_&#x200b;RESOURCE_CREATE** | <code>60m</code> | Maximum time to wait for a runtime resource to be created before considering the step as failed |
| **APP_STEP_TIMEOUTS_&#x200b;CHECK_RUNTIME_&#x200b;RESOURCE_DELETION** | <code>60m</code> | Maximum time to wait for a runtime resource to be deleted before considering the step as failed |
| **APP_STEP_TIMEOUTS_&#x200b;CHECK_RUNTIME_&#x200b;RESOURCE_UPDATE** | <code>180m</code> | Maximum time to wait for a runtime resource to be updated before considering the step as failed |
| **APP_TRIAL_REGION_&#x200b;MAPPING_FILE_PATH** | None | - |
| **APP_UPDATE_MAX_STEP_&#x200b;PROCESSING_TIME** | <code>2m</code> | Maximum time a worker is allowed to process a step before it must return to the update queue |
| **APP_UPDATE_&#x200b;PROCESSING_ENABLED** | <code>true</code> | If true, the broker processes update requests for service instances |
| **APP_UPDATE_WORKERS_&#x200b;AMOUNT** | <code>20</code> | Number of workers in update queue |

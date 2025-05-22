## Kyma Environment Broker Configuration

Kyma Environment Broker (KEB) binary allows you to override some configuration parameters. You can specify the following environment variables:

| Environment Variable | Value | Description |
|---------------------|-------|-------------|
| `APP_ARCHIVING_DRY_RUN` | `True` | If true, runs the archiving process in dry-run mode: no changes are made to the database, only logs what would be archived or deleted. |
| `APP_ARCHIVING_ENABLED` | `False` | If true, enables the archiving mechanism, which stores data about deprovisioned instances in an archive table at the end of the deprovisioning process. |
| `APP_BROKER_ALLOW_UPDATE_EXPIRED_INSTANCE_WITH_CONTEXT` | `false` | Allow update of expired instance |
| `APP_BROKER_BINDING_BINDABLE_PLANS` | `aws` | Comma-separated list of plan names for which service binding is enabled (e.g. "aws,gcp") |
| `APP_BROKER_BINDING_CREATE_BINDING_TIMEOUT` | `15s` | Timeout for creating a binding (e.g. 15s, 1m) |
| `APP_BROKER_BINDING_ENABLED` | `False` | Enable or disable the service binding endpoint (true/false) |
| `APP_BROKER_BINDING_EXPIRATION_SECONDS` | `600` | Default expiration time (in seconds) for a binding if not specified in the request |
| `APP_BROKER_BINDING_MAX_BINDINGS_COUNT` | `10` | Maximum number of non-expired bindings allowed per instance |
| `APP_BROKER_BINDING_MAX_EXPIRATION_SECONDS` | `7200` | Maximum allowed expiration time (in seconds) for a binding |
| `APP_BROKER_BINDING_MIN_EXPIRATION_SECONDS` | `600` | Minimum allowed expiration time (in seconds) for a binding. Can't be lower than 600 seconds. Forced by Gardener |
| `APP_BROKER_DEFAULT_REQUEST_REGION` | `cf-eu10` | Default platform region for requests if not specified (e.g. "cf-eu10") |
| `APP_BROKER_DISABLE_SAP_CONVERGED_CLOUD` | `False` | If true, disables the SAP Converged Cloud plan in the Kyma Environment Broker. When set to true, users cannot provision SAP Converged Cloud clusters |
| `APP_BROKER_ENABLE_PLANS` | `azure,gcp,azure_lite,trial,aws` | Comma-separated list of plan names that are enabled and available for provisioning in the Kyma Environment Broker. |
| `APP_BROKER_ENABLE_SHOOT_AND_SEED_SAME_REGION` | `false` | If true, enforces that the Gardener seed is placed in the same region as the selected shoot region during provisioning. |
| `APP_BROKER_FREE_DOCS_URL` | `https://help.sap.com/docs/` | URL to the documentation for free Kyma environments. Used in API responses and UI labels to direct users to help or documentation about free plans. |
| `APP_BROKER_FREE_EXPIRATION_PERIOD` | `720h` | Used to determine when to show expiration info to users. |
| `APP_BROKER_INCLUDE_ADDITIONAL_PARAMS_IN_SCHEMA` | `false` | If true, additional (advanced or less common) parameters are included in the provisioning schema for service plans. |
| `APP_BROKER_MONITOR_ADDITIONAL_PROPERTIES` | `False` | If true, collects properties from the provisioning request that are not explicitly defined in the schema and stores them in persistent storage. |
| `APP_BROKER_ONLY_ONE_FREE_PER_GA` | `false` | If true, restricts each global account to only one free (freemium) Kyma environment. When enabled, provisioning another free environment for the same global account will be blocked even if the previous one is deprovisioned. |
| `APP_BROKER_ONLY_SINGLE_TRIAL_PER_GA` | `true` | If true, restricts each global account to only one active trial Kyma environment at a time. When enabled, provisioning another trial environment for the same global account will be blocked until the previous one is deprovisioned. |
| `APP_BROKER_OPERATION_TIMEOUT` | `7h` | Maximum allowed duration for processing a single operation (provisioning, deprovisioning, etc.). If the operation exceeds this timeout, it will be marked as failed. Example: "7h" for 7 hours. |
| `APP_BROKER_PORT` | `8080` | Port for the broker HTTP server |
| `APP_BROKER_SHOW_FREE_EXPIRATION_INFO` | `false` | If true, adds expiration information for free plan Kyma environments to API responses and UI labels. |
| `APP_BROKER_SHOW_TRIAL_EXPIRATION_INFO` | `false` | If true, adds expiration information for trial plan Kyma environments to API responses and UI labels. |
| `APP_BROKER_STATUS_PORT` | `8071` | Port for the broker status/health endpoint |
| `APP_BROKER_SUBACCOUNT_MOVEMENT_ENABLED` | `false` | If true, enables subaccount movement (allows changing global account for an instance). |
| `APP_BROKER_SUBACCOUNTS_IDS_TO_SHOW_TRIAL_EXPIRATION_INFO` | `a45be5d8-eddc-4001-91cf-48cc644d571f` | Shows trial expiration information for specific subaccounts in the UI and API responses. |
| `APP_BROKER_TRIAL_DOCS_URL` | `https://help.sap.com/docs/` | URL to the documentation for trial Kyma environments. Used in API responses and UI labels. |
| `APP_BROKER_UPDATE_CUSTOM_RESOURCES_LABELS_ON_ACCOUNT_MOVE` | `false` | If true, updates runtimeCR labels when moving subaccounts |
| `APP_BROKER_URL` | `kyma-env-broker.localhost` | - |
| `APP_BROKER_USE_ADDITIONAL_OIDC_SCHEMA` | `false` | If true, enables the new list-based OIDC schema, allowing multiple OIDC configurations to be specified for a runtime. |
| `APP_CATALOG_FILE_PATH` | - | - |
| `APP_CLEANING_DRY_RUN` | `True` | If true, the cleaning process runs in dry-run mode and does not actually delete any data from the database. |
| `APP_CLEANING_ENABLED` | `False` | If true, enables the cleaning process, which removes all data about deprovisioned instances from the database. |
| `APP_DATABASE_HOST` | - | - |
| `APP_DATABASE_NAME` | - | - |
| `APP_DATABASE_PASSWORD` | - | - |
| `APP_DATABASE_PORT` | - | - |
| `APP_DATABASE_SECRET_KEY` | - | - |
| `APP_DATABASE_SSLMODE` | - | - |
| `APP_DATABASE_SSLROOTCERT` | - | - |
| `APP_DATABASE_USER` | - | - |
| `APP_DISABLE_PROCESS_OPERATIONS_IN_PROGRESS` | `false` | If true, the broker will NOT resume processing operations (provisioning, deprovisioning, updating, etc.) that were in progress when the broker process last stopped or restarted. |
| `APP_DOMAIN_NAME` | `localhost` | - |
| `APP_EDP_ADMIN_URL` | `TBD` | Base URL for the EDP admin API |
| `APP_EDP_AUTH_URL` | `TBD` | OAuth2 token endpoint for EDP |
| `APP_EDP_DISABLED` | `True` | If true, disables EDP integration |
| `APP_EDP_ENVIRONMENT` | `dev` | EDP environment (e.g., dev, prod) |
| `APP_EDP_NAMESPACE` | `kyma-dev` | EDP namespace to use |
| `APP_EDP_REQUIRED` | `False` | If true, EDP integration is required for provisioning |
| `APP_EDP_SECRET` | - | - |
| `APP_EVENTS_ENABLED` | `True` | Enable or disable the /events API and event storage for operation events (true/false) |
| `APP_FREEMIUM_WHITELISTED_GLOBAL_ACCOUNTS_FILE_PATH` | - | - |
| `APP_GARDENER_KUBECONFIG_PATH` | `/gardener/kubeconfig/kubeconfig` | Path to the kubeconfig file for accessing the Gardener cluster. |
| `APP_GARDENER_PROJECT` | `kyma-dev` | Gardener project connected to SA for HAP credentials lookup. |
| `APP_GARDENER_SHOOT_DOMAIN` | `kyma-dev.shoot.canary.k8s-hana.ondemand.com` | Default domain for shoots (clusters) created by Gardener. |
| `APP_HAP_RULE_FILE_PATH` | - | - |
| `APP_INFRASTRUCTURE_MANAGER_CONTROL_PLANE_FAILURE_TOLERANCE` | - | Sets the failure tolerance level for the Kubernetes control plane in Gardener clusters. Possible values: "node", "zone", or empty (default). |
| `APP_INFRASTRUCTURE_MANAGER_DEFAULT_GARDENER_SHOOT_PURPOSE` | `development` | Sets the default purpose for Gardener shoots (clusters) created by the broker. Possible values: development, evaluation, production, testing |
| `APP_INFRASTRUCTURE_MANAGER_DEFAULT_TRIAL_PROVIDER` | `Azure` | Sets the default cloud provider to use for trial Kyma environments (e.g., Azure, AWS). |
| `APP_INFRASTRUCTURE_MANAGER_ENABLE_INGRESS_FILTERING` | `false` | If true, allows to enable ingress filtering for defined plans. |
| `APP_INFRASTRUCTURE_MANAGER_INGRESS_FILTERING_PLANS` | `azure,gcp,aws` | Comma-separated list of plan names for which ingress filtering is available. |
| `APP_INFRASTRUCTURE_MANAGER_KUBERNETES_VERSION` | `1.16.9` | Sets the default Kubernetes version to use for new clusters provisioned by the broker. |
| `APP_INFRASTRUCTURE_MANAGER_MACHINE_IMAGE` | - | Sets the default machine image name to use for nodes in provisioned clusters. If empty, the Gardener default value is used. |
| `APP_INFRASTRUCTURE_MANAGER_MACHINE_IMAGE_VERSION` | - | Sets the version of the machine image to use for nodes in provisioned clusters. If empty, the Gardener default value is used. |
| `APP_INFRASTRUCTURE_MANAGER_MULTI_ZONE_CLUSTER` | `false` | If true, enables provisioning of clusters with nodes distributed across multiple availability zones. |
| `APP_INFRASTRUCTURE_MANAGER_USE_SMALLER_MACHINE_TYPES` | `false` | If true, provisions trial, freemium, and azure_lite clusters using smaller machine types. |
| `APP_KUBECONFIG_ALLOW_ORIGINS` | `*` | Specifies which origins are allowed for CORS (Cross-Origin Resource Sharing) on the /kubeconfig endpoint. |
| `APP_KYMA_DASHBOARD_CONFIG_LANDSCAPE_URL` | `https://dashboard.dev.kyma.cloud.sap` | The base URL of the Kyma Dashboard used to generate links to the web UI for Kyma environments. |
| `APP_LIFECYCLE_MANAGER_INTEGRATION_DISABLED` | `false` | When disabled, the broker will not create, update, or delete the KymaCR. |
| `APP_METRICSV2_ENABLED` | `False` | If true, enables metricsv2 collection and Prometheus exposure. |
| `APP_METRICSV2_OPERATION_RESULT_FINISHED_OPERATION_RETENTION_PERIOD` | `3h` | How long to retain finished operation results in memory (e.g., 3h). |
| `APP_METRICSV2_OPERATION_RESULT_POLLING_INTERVAL` | `1m` | How often to poll for operation results (e.g., 1m). |
| `APP_METRICSV2_OPERATION_RESULT_RETENTION_PERIOD` | `1h` | How long to retain operation results (e.g., 1h). |
| `APP_METRICSV2_OPERATION_STATS_POLLING_INTERVAL` | `1m` | How often to poll for operation statistics (e.g., 1m). |
| `APP_MULTIPLE_CONTEXTS` | `False` | If true, generates kubeconfig files with multiple contexts (if possible) instead of a single context. |
| `APP_PLANS_CONFIGURATION_FILE_PATH` | - | - |
| `APP_PROFILER_MEMORY` | `False` | Enable memory profiler (true/false) |
| `APP_PROVIDERS_CONFIGURATION_FILE_PATH` | - | - |
| `APP_REGIONS_SUPPORTING_MACHINE_FILE_PATH` | - | - |
| `APP_RUNTIME_CONFIGURATION_CONFIG_MAP_NAME` | - | - |
| `APP_SAP_CONVERGED_CLOUD_REGION_MAPPINGS_FILE_PATH` | - | - |
| `APP_SKR_DNS_PROVIDERS_VALUES_YAML_FILE_PATH` | - | - |
| `APP_SKR_OIDC_DEFAULT_VALUES_YAML_FILE_PATH` | - | - |
| `APP_STEP_TIMEOUTS_CHECK_RUNTIME_RESOURCE_CREATE` | `60m` | Maximum time to wait for a runtime resource to be created before considering the step as failed (e.g., 60m = 60 minutes). |
| `APP_STEP_TIMEOUTS_CHECK_RUNTIME_RESOURCE_DELETION` | `60m` | Maximum time to wait for a runtime resource to be deleted before considering the step as failed (e.g., 60m = 60 minutes). |
| `APP_STEP_TIMEOUTS_CHECK_RUNTIME_RESOURCE_UPDATE` | `180m` | Maximum time to wait for a runtime resource to be updated before considering the step as failed (e.g., 180m = 180 minutes). |
| `APP_TRIAL_REGION_MAPPING_FILE_PATH` | - | - |
| `APP_UPDATE_PROCESSING_ENABLED` | `true` | If true, the broker processes update requests for service instances |

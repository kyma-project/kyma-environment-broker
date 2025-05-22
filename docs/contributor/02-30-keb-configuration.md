## Kyma Environment Broker Configuration

Kyma Environment Broker (KEB) binary allows you to override some configuration parameters. You can specify the following environment variables:

| Environment Variable | Current Value | Description |
|---------------------|-------|-------------|
| `APP_ARCHIVING_DRY_RUN` | `true` | If true, runs the archiving process in dry-run mode: Makes no changes to the database, only logs what is to be archived or deleted |
| `APP_ARCHIVING_ENABLED` | `false` | If true, enables the archiving mechanism, which stores data about deprovisioned instances in an archive table at the end of the deprovisioning process |
| `APP_BROKER_ALLOW_UPDATE_EXPIRED_INSTANCE_WITH_CONTEXT` | `false` | Allow update of expired instance |
| `APP_BROKER_BINDING_BINDABLE_PLANS` | `aws` | Comma-separated list of plan names for which service binding is enabled, for example, "aws,gcp" |
| `APP_BROKER_BINDING_CREATE_BINDING_TIMEOUT` | `15s` | Timeout for creating a binding, for example, 15s, 1m |
| `APP_BROKER_BINDING_ENABLED` | `false` | Enables or disables the service binding endpoint (true/false) |
| `APP_BROKER_BINDING_EXPIRATION_SECONDS` | `600` | Default expiration time (in seconds) for a binding if not specified in the request |
| `APP_BROKER_BINDING_MAX_BINDINGS_COUNT` | `10` | Maximum number of non-expired bindings allowed per instance |
| `APP_BROKER_BINDING_MAX_EXPIRATION_SECONDS` | `7200` | Maximum allowed expiration time (in seconds) for a binding |
| `APP_BROKER_BINDING_MIN_EXPIRATION_SECONDS` | `600` | Minimum allowed expiration time (in seconds) for a binding. Can't be lower than 600 seconds. Forced by Gardener |
| `APP_BROKER_DEFAULT_REQUEST_REGION` | `cf-eu10` | Default platform region for requests if not specified |
| `APP_BROKER_DISABLE_SAP_CONVERGED_CLOUD` | `false` | If true, disables the SAP Cloud Infrastructure plan in the KEB. When set to true, users cannot provision SAP Cloud Infrastructure clusters |
| `APP_BROKER_ENABLE_PLANS` | `azure,gcp,azure_lite,trial,aws` | Comma-separated list of plan names enabled and available for provisioning in KEB |
| `APP_BROKER_ENABLE_SHOOT_AND_SEED_SAME_REGION` | `false` | If true, enforces that the Gardener seed is placed in the same region as the shoot region selected during provisioning |
| `APP_BROKER_FREE_DOCS_URL` | `https://help.sap.com/docs/` | URL to the documentation of free Kyma runtimes. Used in API responses and UI labels to direct users to help or documentation about free plans |
| `APP_BROKER_FREE_EXPIRATION_PERIOD` | `720h` | Determines when to show expiration info to users |
| `APP_BROKER_INCLUDE_ADDITIONAL_PARAMS_IN_SCHEMA` | `false` | If true, additional (advanced or less common) parameters are included in the provisioning schema for service plans |
| `APP_BROKER_MONITOR_ADDITIONAL_PROPERTIES` | `false` | If true, collects properties from the provisioning request that are not explicitly defined in the schema and stores them in persistent storage |
| `APP_BROKER_ONLY_ONE_FREE_PER_GA` | `false` | If true, restricts each global account to only one free (freemium) Kyma runtime. When enabled, provisioning another free environment for the same global account is blocked even if the previous one is deprovisioned |
| `APP_BROKER_ONLY_SINGLE_TRIAL_PER_GA` | `true` | If true, restricts each global account to only one active trial Kyma runtime at a time When enabled, provisioning another trial environment for the same global account is blocked until the previous one is deprovisioned |
| `APP_BROKER_OPERATION_TIMEOUT` | `7h` | Maximum allowed duration for processing a single operation (provisioning, deprovisioning, etc.) If the operation exceeds this timeout, it is marked as failed. Example: "7h" for 7 hours |
| `APP_BROKER_PORT` | `8080` | Port for the broker HTTP server |
| `APP_BROKER_SHOW_FREE_EXPIRATION_INFO` | `false` | If true, adds expiration information for free plan Kyma runtimes to API responses and UI labels |
| `APP_BROKER_SHOW_TRIAL_EXPIRATION_INFO` | `false` | If true, adds expiration information for trial plan Kyma runtimes to API responses and UI labels |
| `APP_BROKER_STATUS_PORT` | `8071` | Port for the broker status/health endpoint |
| `APP_BROKER_SUBACCOUNT_MOVEMENT_ENABLED` | `false` | If true, enables subaccount movement (allows changing global account for an instance) |
| `APP_BROKER_SUBACCOUNTS_IDS_TO_SHOW_TRIAL_EXPIRATION_INFO` | `a45be5d8-eddc-4001-91cf-48cc644d571f` | Shows trial expiration information for specific subaccounts in the UI and API responses |
| `APP_BROKER_TRIAL_DOCS_URL` | `https://help.sap.com/docs/` | URL to the documentation for trial Kyma runtimes. Used in API responses and UI labels |
| `APP_BROKER_UPDATE_CUSTOM_RESOURCES_LABELS_ON_ACCOUNT_MOVE` | `false` | If true, updates runtimeCR labels when moving subaccounts |
| `APP_BROKER_URL` | `kyma-env-broker.localhost` | - |
| `APP_BROKER_USE_ADDITIONAL_OIDC_SCHEMA` | `false` | If true, enables the new list-based OIDC schema, allowing multiple OIDC configurations for a runtime |
| `APP_CATALOG_FILE_PATH` | None | - |
| `APP_CLEANING_DRY_RUN` | `true` | If true, the cleaning process runs in dry-run mode and does not actually delete any data from the database |
| `APP_CLEANING_ENABLED` | `false` | If true, enables the cleaning process, which removes all data about deprovisioned instances from the database |
| `APP_DATABASE_HOST` | None | - |
| `APP_DATABASE_NAME` | None | - |
| `APP_DATABASE_PASSWORD` | None | - |
| `APP_DATABASE_PORT` | None | - |
| `APP_DATABASE_SECRET_KEY` | None | - |
| `APP_DATABASE_SSLMODE` | None | - |
| `APP_DATABASE_SSLROOTCERT` | None | - |
| `APP_DATABASE_USER` | None | - |
| `APP_DISABLE_PROCESS_OPERATIONS_IN_PROGRESS` | `false` | If true, the broker does NOT resume processing operations (provisioning, deprovisioning, updating, etc.) that were in progress when the broker process last stopped or restarted |
| `APP_DOMAIN_NAME` | `localhost` | - |
| `APP_EDP_ADMIN_URL` | `TBD` | Base URL for the EDP admin API |
| `APP_EDP_AUTH_URL` | `TBD` | OAuth2 token endpoint for EDP |
| `APP_EDP_DISABLED` | `true` | If true, disables EDP integration |
| `APP_EDP_ENVIRONMENT` | `dev` | EDP environment, for example, dev, prod |
| `APP_EDP_NAMESPACE` | `kyma-dev` | EDP namespace to use |
| `APP_EDP_REQUIRED` | `false` | If true, EDP integration is required for provisioning |
| `APP_EDP_SECRET` | None | - |
| `APP_EVENTS_ENABLED` | `true` | Enables or disables the /events API and event storage for operation events (true/false) |
| `APP_FREEMIUM_WHITELISTED_GLOBAL_ACCOUNTS_FILE_PATH` | None | - |
| `APP_GARDENER_KUBECONFIG_PATH` | `/gardener/kubeconfig/kubeconfig` | Path to the kubeconfig file for accessing the Gardener cluster |
| `APP_GARDENER_PROJECT` | `kyma-dev` | Gardener project connected to SA for HAP credentials lookup |
| `APP_GARDENER_SHOOT_DOMAIN` | `kyma-dev.shoot.canary.k8s-hana.ondemand.com` | Default domain for shoots (clusters) created by Gardener |
| `APP_HAP_RULE_FILE_PATH` | None | - |
| `APP_INFRASTRUCTURE_MANAGER_CONTROL_PLANE_FAILURE_TOLERANCE` | None | Sets the failure tolerance level for the Kubernetes control plane in Gardener clusters Possible values: empty (default), "node", or "zone" |
| `APP_INFRASTRUCTURE_MANAGER_DEFAULT_GARDENER_SHOOT_PURPOSE` | `development` | Sets the default purpose for Gardener shoots (clusters) created by the broker Possible values: development, evaluation, production, testing |
| `APP_INFRASTRUCTURE_MANAGER_DEFAULT_TRIAL_PROVIDER` | `Azure` | Sets the default cloud provider for trial Kyma runtimes, for example, Azure, AWS |
| `APP_INFRASTRUCTURE_MANAGER_ENABLE_INGRESS_FILTERING` | `false` | If true, enables ingress filtering for defined plans |
| `APP_INFRASTRUCTURE_MANAGER_INGRESS_FILTERING_PLANS` | `azure,gcp,aws` | Comma-separated list of plan names for which ingress filtering is available |
| `APP_INFRASTRUCTURE_MANAGER_KUBERNETES_VERSION` | `1.16.9` | Sets the default Kubernetes version for new clusters provisioned by the broker |
| `APP_INFRASTRUCTURE_MANAGER_MACHINE_IMAGE` | None | Sets the default machine image name for nodes in provisioned clusters. If empty, the Gardener default value is used |
| `APP_INFRASTRUCTURE_MANAGER_MACHINE_IMAGE_VERSION` | None | Sets the version of the machine image for nodes in provisioned clusters. If empty, the Gardener default value is used |
| `APP_INFRASTRUCTURE_MANAGER_MULTI_ZONE_CLUSTER` | `false` | If true, enables provisioning of clusters with nodes distributed across multiple availability zones |
| `APP_INFRASTRUCTURE_MANAGER_USE_SMALLER_MACHINE_TYPES` | `false` | If true, provisions trial, freemium, and azure_lite clusters using smaller machine types |
| `APP_KUBECONFIG_ALLOW_ORIGINS` | `*` | Specifies which origins are allowed for Cross-Origin Resource Sharing (CORS) on the /kubeconfig endpoint |
| `APP_KYMA_DASHBOARD_CONFIG_LANDSCAPE_URL` | `https://dashboard.dev.kyma.cloud.sap` | The base URL of the Kyma Dashboard used to generate links to the web UI for Kyma runtimes |
| `APP_LIFECYCLE_MANAGER_INTEGRATION_DISABLED` | `false` | When disabled, the broker does not create, update, or delete the KymaCR |
| `APP_METRICSV2_ENABLED` | `false` | If true, enables metricsv2 collection and Prometheus exposure |
| `APP_METRICSV2_OPERATION_RESULT_FINISHED_OPERATION_RETENTION_PERIOD` | `3h` | Duration of retaining finished operation results in memory |
| `APP_METRICSV2_OPERATION_RESULT_POLLING_INTERVAL` | `1m` | Frequency of polling for operation results |
| `APP_METRICSV2_OPERATION_RESULT_RETENTION_PERIOD` | `1h` | Duration of retaining operation results |
| `APP_METRICSV2_OPERATION_STATS_POLLING_INTERVAL` | `1m` | Frequency of polling for operation statistics |
| `APP_MULTIPLE_CONTEXTS` | `false` | If true, generates kubeconfig files with multiple contexts (if possible) instead of a single context |
| `APP_PLANS_CONFIGURATION_FILE_PATH` | None | - |
| `APP_PROFILER_MEMORY` | `false` | Enables memory profiler (true/false) |
| `APP_PROVIDERS_CONFIGURATION_FILE_PATH` | None | - |
| `APP_REGIONS_SUPPORTING_MACHINE_FILE_PATH` | None | - |
| `APP_RUNTIME_CONFIGURATION_CONFIG_MAP_NAME` | None | - |
| `APP_SAP_CONVERGED_CLOUD_REGION_MAPPINGS_FILE_PATH` | None | - |
| `APP_SKR_DNS_PROVIDERS_VALUES_YAML_FILE_PATH` | None | - |
| `APP_SKR_OIDC_DEFAULT_VALUES_YAML_FILE_PATH` | None | - |
| `APP_STEP_TIMEOUTS_CHECK_RUNTIME_RESOURCE_CREATE` | `60m` | Maximum time to wait for a runtime resource to be created before considering the step as failed |
| `APP_STEP_TIMEOUTS_CHECK_RUNTIME_RESOURCE_DELETION` | `60m` | Maximum time to wait for a runtime resource to be deleted before considering the step as failed |
| `APP_STEP_TIMEOUTS_CHECK_RUNTIME_RESOURCE_UPDATE` | `180m` | Maximum time to wait for a runtime resource to be updated before considering the step as failed |
| `APP_TRIAL_REGION_MAPPING_FILE_PATH` | None | - |
| `APP_UPDATE_PROCESSING_ENABLED` | `true` | If true, the broker processes update requests for service instances |

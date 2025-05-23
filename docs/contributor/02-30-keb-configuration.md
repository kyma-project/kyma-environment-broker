## Kyma Environment Broker Configuration

Kyma Environment Broker (KEB) binary allows you to override some configuration parameters. You can specify the following environment variables:

| Environment Variable | Current Value | Description |
|---------------------|------------------------------|---------------------------------------------------------------|
| <code>APP_ARCHIVING_DRY_&#x200b;RUN</code> | <code>true</code> | If true, runs the archiving process in dry-run mode: Makes no changes to the database, only logs what is to be archived or deleted |
| <code>APP_ARCHIVING_&#x200b;ENABLED</code> | <code>false</code> | If true, enables the archiving mechanism, which stores data about deprovisioned instances in an archive table at the end of the deprovisioning process |
| <code>APP_BROKER_ALLOW_&#x200b;UPDATE_EXPIRED_&#x200b;INSTANCE_WITH_&#x200b;CONTEXT</code> | <code>false</code> | Allow update of expired instance |
| <code>APP_BROKER_BINDING_&#x200b;BINDABLE_PLANS</code> | <code>aws</code> | Comma-separated list of plan names for which service binding is enabled, for example, "aws,gcp" |
| <code>APP_BROKER_BINDING_&#x200b;CREATE_BINDING_&#x200b;TIMEOUT</code> | <code>15s</code> | Timeout for creating a binding, for example, 15s, 1m |
| <code>APP_BROKER_BINDING_&#x200b;ENABLED</code> | <code>false</code> | Enables or disables the service binding endpoint (true/false) |
| <code>APP_BROKER_BINDING_&#x200b;EXPIRATION_SECONDS</code> | <code>600</code> | Default expiration time (in seconds) for a binding if not specified in the request |
| <code>APP_BROKER_BINDING_&#x200b;MAX_BINDINGS_COUNT</code> | <code>10</code> | Maximum number of non-expired bindings allowed per instance |
| <code>APP_BROKER_BINDING_&#x200b;MAX_EXPIRATION_&#x200b;SECONDS</code> | <code>7200</code> | Maximum allowed expiration time (in seconds) for a binding |
| <code>APP_BROKER_BINDING_&#x200b;MIN_EXPIRATION_&#x200b;SECONDS</code> | <code>600</code> | Minimum allowed expiration time (in seconds) for a binding. Can't be lower than 600 seconds. Forced by Gardener |
| <code>APP_BROKER_DEFAULT_&#x200b;REQUEST_REGION</code> | <code>cf-eu10</code> | Default platform region for requests if not specified |
| <code>APP_BROKER_DISABLE_&#x200b;SAP_CONVERGED_CLOUD</code> | <code>false</code> | If true, disables the SAP Cloud Infrastructure plan in the KEB. When set to true, users cannot provision SAP Cloud Infrastructure clusters |
| <code>APP_BROKER_ENABLE_&#x200b;PLANS</code> | <code>azure,gcp,&#x200b;azure_lite&#x200b;,trial,aws</code> | Comma-separated list of plan names enabled and available for provisioning in KEB |
| <code>APP_BROKER_ENABLE_&#x200b;SHOOT_AND_SEED_SAME_&#x200b;REGION</code> | <code>false</code> | If true, enforces that the Gardener seed is placed in the same region as the shoot region selected during provisioning |
| <code>APP_BROKER_FREE_&#x200b;DOCS_URL</code> | <code>https://he&#x200b;lp.sap.com&#x200b;/docs/</code> | URL to the documentation of free Kyma runtimes. Used in API responses and UI labels to direct users to help or documentation about free plans |
| <code>APP_BROKER_FREE_&#x200b;EXPIRATION_PERIOD</code> | <code>720h</code> | Determines when to show expiration info to users |
| <code>APP_BROKER_INCLUDE_&#x200b;ADDITIONAL_PARAMS_&#x200b;IN_SCHEMA</code> | <code>false</code> | If true, additional (advanced or less common) parameters are included in the provisioning schema for service plans |
| <code>APP_BROKER_MONITOR_&#x200b;ADDITIONAL_&#x200b;PROPERTIES</code> | <code>false</code> | If true, collects properties from the provisioning request that are not explicitly defined in the schema and stores them in persistent storage |
| <code>APP_BROKER_ONLY_ONE_&#x200b;FREE_PER_GA</code> | <code>false</code> | If true, restricts each global account to only one free (freemium) Kyma runtime. When enabled, provisioning another free environment for the same global account is blocked even if the previous one is deprovisioned |
| <code>APP_BROKER_ONLY_&#x200b;SINGLE_TRIAL_PER_GA</code> | <code>true</code> | If true, restricts each global account to only one active trial Kyma runtime at a time When enabled, provisioning another trial environment for the same global account is blocked until the previous one is deprovisioned |
| <code>APP_BROKER_&#x200b;OPERATION_TIMEOUT</code> | <code>7h</code> | Maximum allowed duration for processing a single operation (provisioning, deprovisioning, etc.) If the operation exceeds this timeout, it is marked as failed. Example: "7h" for 7 hours |
| <code>APP_BROKER_PORT</code> | <code>8080</code> | Port for the broker HTTP server |
| <code>APP_BROKER_SHOW_&#x200b;FREE_EXPIRATION_INFO</code> | <code>false</code> | If true, adds expiration information for free plan Kyma runtimes to API responses and UI labels |
| <code>APP_BROKER_SHOW_&#x200b;TRIAL_EXPIRATION_&#x200b;INFO</code> | <code>false</code> | If true, adds expiration information for trial plan Kyma runtimes to API responses and UI labels |
| <code>APP_BROKER_STATUS_&#x200b;PORT</code> | <code>8071</code> | Port for the broker status/health endpoint |
| <code>APP_BROKER_&#x200b;SUBACCOUNT_MOVEMENT_&#x200b;ENABLED</code> | <code>false</code> | If true, enables subaccount movement (allows changing global account for an instance) |
| <code>APP_BROKER_&#x200b;SUBACCOUNTS_IDS_TO_&#x200b;SHOW_TRIAL_&#x200b;EXPIRATION_INFO</code> | <code>a45be5d8-e&#x200b;ddc-4001-9&#x200b;1cf-48cc64&#x200b;4d571f</code> | Shows trial expiration information for specific subaccounts in the UI and API responses |
| <code>APP_BROKER_TRIAL_&#x200b;DOCS_URL</code> | <code>https://he&#x200b;lp.sap.com&#x200b;/docs/</code> | URL to the documentation for trial Kyma runtimes. Used in API responses and UI labels |
| <code>APP_BROKER_UPDATE_&#x200b;CUSTOM_RESOURCES_&#x200b;LABELS_ON_ACCOUNT_&#x200b;MOVE</code> | <code>false</code> | If true, updates runtimeCR labels when moving subaccounts |
| <code>APP_BROKER_URL</code> | <code>kyma-env-b&#x200b;roker.loca&#x200b;lhost</code> | - |
| <code>APP_BROKER_USE_&#x200b;ADDITIONAL_OIDC_&#x200b;SCHEMA</code> | <code>false</code> | If true, enables the new list-based OIDC schema, allowing multiple OIDC configurations for a runtime |
| <code>APP_CATALOG_FILE_&#x200b;PATH</code> | None | - |
| <code>APP_CLEANING_DRY_RUN</code> | <code>true</code> | If true, the cleaning process runs in dry-run mode and does not actually delete any data from the database |
| <code>APP_CLEANING_ENABLED</code> | <code>false</code> | If true, enables the cleaning process, which removes all data about deprovisioned instances from the database |
| <code>APP_DATABASE_HOST</code> | None | - |
| <code>APP_DATABASE_NAME</code> | None | - |
| <code>APP_DATABASE_&#x200b;PASSWORD</code> | None | - |
| <code>APP_DATABASE_PORT</code> | None | - |
| <code>APP_DATABASE_SECRET_&#x200b;KEY</code> | None | - |
| <code>APP_DATABASE_SSLMODE</code> | None | - |
| <code>APP_DATABASE_&#x200b;SSLROOTCERT</code> | None | - |
| <code>APP_DATABASE_USER</code> | None | - |
| <code>APP_DISABLE_PROCESS_&#x200b;OPERATIONS_IN_&#x200b;PROGRESS</code> | <code>false</code> | If true, the broker does NOT resume processing operations (provisioning, deprovisioning, updating, etc.) that were in progress when the broker process last stopped or restarted |
| <code>APP_DOMAIN_NAME</code> | <code>localhost</code> | - |
| <code>APP_EDP_ADMIN_URL</code> | <code>TBD</code> | Base URL for the EDP admin API |
| <code>APP_EDP_AUTH_URL</code> | <code>TBD</code> | OAuth2 token endpoint for EDP |
| <code>APP_EDP_DISABLED</code> | <code>true</code> | If true, disables EDP integration |
| <code>APP_EDP_ENVIRONMENT</code> | <code>dev</code> | EDP environment, for example, dev, prod |
| <code>APP_EDP_NAMESPACE</code> | <code>kyma-dev</code> | EDP namespace to use |
| <code>APP_EDP_REQUIRED</code> | <code>false</code> | If true, EDP integration is required for provisioning |
| <code>APP_EDP_SECRET</code> | None | - |
| <code>APP_EVENTS_ENABLED</code> | <code>true</code> | Enables or disables the /events API and event storage for operation events (true/false) |
| <code>APP_FREEMIUM_&#x200b;WHITELISTED_GLOBAL_&#x200b;ACCOUNTS_FILE_PATH</code> | None | - |
| <code>APP_GARDENER_&#x200b;KUBECONFIG_PATH</code> | <code>/gardener/&#x200b;kubeconfig&#x200b;/kubeconfi&#x200b;g</code> | Path to the kubeconfig file for accessing the Gardener cluster |
| <code>APP_GARDENER_PROJECT</code> | <code>kyma-dev</code> | Gardener project connected to SA for HAP credentials lookup |
| <code>APP_GARDENER_SHOOT_&#x200b;DOMAIN</code> | <code>kyma-dev.s&#x200b;hoot.canar&#x200b;y.k8s-hana&#x200b;.ondemand.&#x200b;com</code> | Default domain for shoots (clusters) created by Gardener |
| <code>APP_HAP_RULE_FILE_&#x200b;PATH</code> | None | - |
| <code>APP_INFRASTRUCTURE_&#x200b;MANAGER_CONTROL_&#x200b;PLANE_FAILURE_&#x200b;TOLERANCE</code> | None | Sets the failure tolerance level for the Kubernetes control plane in Gardener clusters Possible values: empty (default), "node", or "zone" |
| <code>APP_INFRASTRUCTURE_&#x200b;MANAGER_DEFAULT_&#x200b;GARDENER_SHOOT_&#x200b;PURPOSE</code> | <code>developmen&#x200b;t</code> | Sets the default purpose for Gardener shoots (clusters) created by the broker Possible values: development, evaluation, production, testing |
| <code>APP_INFRASTRUCTURE_&#x200b;MANAGER_DEFAULT_&#x200b;TRIAL_PROVIDER</code> | <code>Azure</code> | Sets the default cloud provider for trial Kyma runtimes, for example, Azure, AWS |
| <code>APP_INFRASTRUCTURE_&#x200b;MANAGER_ENABLE_&#x200b;INGRESS_FILTERING</code> | <code>false</code> | If true, enables ingress filtering for defined plans |
| <code>APP_INFRASTRUCTURE_&#x200b;MANAGER_INGRESS_&#x200b;FILTERING_PLANS</code> | <code>azure,gcp,&#x200b;aws</code> | Comma-separated list of plan names for which ingress filtering is available |
| <code>APP_INFRASTRUCTURE_&#x200b;MANAGER_KUBERNETES_&#x200b;VERSION</code> | <code>1.16.9</code> | Sets the default Kubernetes version for new clusters provisioned by the broker |
| <code>APP_INFRASTRUCTURE_&#x200b;MANAGER_MACHINE_&#x200b;IMAGE</code> | None | Sets the default machine image name for nodes in provisioned clusters. If empty, the Gardener default value is used |
| <code>APP_INFRASTRUCTURE_&#x200b;MANAGER_MACHINE_&#x200b;IMAGE_VERSION</code> | None | Sets the version of the machine image for nodes in provisioned clusters. If empty, the Gardener default value is used |
| <code>APP_INFRASTRUCTURE_&#x200b;MANAGER_MULTI_ZONE_&#x200b;CLUSTER</code> | <code>false</code> | If true, enables provisioning of clusters with nodes distributed across multiple availability zones |
| <code>APP_INFRASTRUCTURE_&#x200b;MANAGER_USE_SMALLER_&#x200b;MACHINE_TYPES</code> | <code>false</code> | If true, provisions trial, freemium, and azure_lite clusters using smaller machine types |
| <code>APP_KUBECONFIG_&#x200b;ALLOW_ORIGINS</code> | <code>*</code> | Specifies which origins are allowed for Cross-Origin Resource Sharing (CORS) on the /kubeconfig endpoint |
| <code>APP_KYMA_DASHBOARD_&#x200b;CONFIG_LANDSCAPE_URL</code> | <code>https://da&#x200b;shboard.de&#x200b;v.kyma.clo&#x200b;ud.sap</code> | The base URL of the Kyma Dashboard used to generate links to the web UI for Kyma runtimes |
| <code>APP_LIFECYCLE_&#x200b;MANAGER_INTEGRATION_&#x200b;DISABLED</code> | <code>false</code> | When disabled, the broker does not create, update, or delete the KymaCR |
| <code>APP_METRICSV2_&#x200b;ENABLED</code> | <code>false</code> | If true, enables metricsv2 collection and Prometheus exposure |
| <code>APP_METRICSV2_&#x200b;OPERATION_RESULT_&#x200b;FINISHED_OPERATION_&#x200b;RETENTION_PERIOD</code> | <code>3h</code> | Duration of retaining finished operation results in memory |
| <code>APP_METRICSV2_&#x200b;OPERATION_RESULT_&#x200b;POLLING_INTERVAL</code> | <code>1m</code> | Frequency of polling for operation results |
| <code>APP_METRICSV2_&#x200b;OPERATION_RESULT_&#x200b;RETENTION_PERIOD</code> | <code>1h</code> | Duration of retaining operation results |
| <code>APP_METRICSV2_&#x200b;OPERATION_STATS_&#x200b;POLLING_INTERVAL</code> | <code>1m</code> | Frequency of polling for operation statistics |
| <code>APP_MULTIPLE_&#x200b;CONTEXTS</code> | <code>false</code> | If true, generates kubeconfig files with multiple contexts (if possible) instead of a single context |
| <code>APP_PLANS_&#x200b;CONFIGURATION_FILE_&#x200b;PATH</code> | None | - |
| <code>APP_PROFILER_MEMORY</code> | <code>false</code> | Enables memory profiler (true/false) |
| <code>APP_PROVIDERS_&#x200b;CONFIGURATION_FILE_&#x200b;PATH</code> | None | - |
| <code>APP_REGIONS_&#x200b;SUPPORTING_MACHINE_&#x200b;FILE_PATH</code> | None | - |
| <code>APP_RUNTIME_&#x200b;CONFIGURATION_&#x200b;CONFIG_MAP_NAME</code> | None | - |
| <code>APP_SAP_CONVERGED_&#x200b;CLOUD_REGION_&#x200b;MAPPINGS_FILE_PATH</code> | None | - |
| <code>APP_SKR_DNS_&#x200b;PROVIDERS_VALUES_&#x200b;YAML_FILE_PATH</code> | None | - |
| <code>APP_SKR_OIDC_&#x200b;DEFAULT_VALUES_YAML_&#x200b;FILE_PATH</code> | None | - |
| <code>APP_STEP_TIMEOUTS_&#x200b;CHECK_RUNTIME_&#x200b;RESOURCE_CREATE</code> | <code>60m</code> | Maximum time to wait for a runtime resource to be created before considering the step as failed |
| <code>APP_STEP_TIMEOUTS_&#x200b;CHECK_RUNTIME_&#x200b;RESOURCE_DELETION</code> | <code>60m</code> | Maximum time to wait for a runtime resource to be deleted before considering the step as failed |
| <code>APP_STEP_TIMEOUTS_&#x200b;CHECK_RUNTIME_&#x200b;RESOURCE_UPDATE</code> | <code>180m</code> | Maximum time to wait for a runtime resource to be updated before considering the step as failed |
| <code>APP_TRIAL_REGION_&#x200b;MAPPING_FILE_PATH</code> | None | - |
| <code>APP_UPDATE_&#x200b;PROCESSING_ENABLED</code> | <code>true</code> | If true, the broker processes update requests for service instances |

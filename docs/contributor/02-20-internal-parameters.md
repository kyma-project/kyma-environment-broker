<!--{"metadata":{"publish":false}}-->

# Internal Provisioning Parameters

These parameters belong to the `ProvisioningParameters` struct in [`internal/dto.go`](../../internal/dto.go) and its embedded `ProvisioningParametersDTO` sub-struct in [`common/runtime/model.go`](../../common/runtime/model.go). They are used internally by Kyma Environment Broker (KEB) but are not exposed to users using the Open Service Broker (OSB) provisioning API.

For user-facing provisioning parameters, see [Service Description](../user/03-10-service-description.md).

| Parameter | Type | Description |
|---|---|---|
| **purpose** | string | Provides a purpose for a Kyma runtime. |
| **targetSecret** | string | Name of the Secret containing hyperscaler credentials. |
| **platform_region** | string | Platform region sent in the request path. |
| **platform_provider** | string | Platform provider for a Kyma runtime. |
| **context.tenant_id** | string | Tenant ID for a Kyma runtime. |
| **context.subaccount_id** | string | Subaccount ID for a Kyma runtime. |
| **context.globalaccount_id** | string | Global account ID for a Kyma runtime. |
| **context.sm_operator_credentials.clientid** | string | Client ID for the SAP BTP service operator. |
| **context.sm_operator_credentials.clientsecret** | string | Client secret for the SAP BTP service operator. |
| **context.sm_operator_credentials.sm_url** | string | SAP Service Manager URL for the SAP BTP service operator. |
| **context.sm_operator_credentials.url** | string | Authentication URL for the SAP BTP service operator. |
| **context.sm_operator_credentials.xsappname** | string | XSApp name for the SAP BTP service operator. |
| **context.user_id** | string | User ID for a Kyma runtime. |
| **volumeSizeGb** | int | Root volume size in GB. Set by the provider configuration, not user-configurable. |
| **zones** | []string | Availability zones. Set by the provider configuration, not user-configurable. |
| **shootName** | string | Name of the Gardener Shoot resource. |
| **shootDomain** | string | Domain of the Gardener Shoot resource. |
| **kubeconfig** | string | Kubeconfig for the Kyma runtime. |
| **maxSurge** | int | Maximum number of virtual machines created during an update. |
| **maxUnavailable** | int | Maximum number of virtual machines unavailable during an update. |

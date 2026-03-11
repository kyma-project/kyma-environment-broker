# Hyperscaler Account Pool

> ### Note:
> The feature referred to as Hyperscaler Account Pool (HAP) manages entities that are identified as `SubscriptionSecrets` in code. 
> You may encounter this terminology in code references such as `ResolveSubscriptionSecretStep` and `SubscriptionSecretName`.

To provision clusters through Gardener using Kyma Infrastructure Manager (KIM), Kyma Environment Broker (KEB) requires a hyperscaler (Google Cloud, Microsoft Azure, Amazon Web Services, etc.) account/subscription. Managing the available hyperscaler accounts is outside the scope of KEB. Instead, the available accounts are handled by HAP.

HAP stores credentials for the hyperscaler accounts that have been set up in advance in Kubernetes Secrets. The credentials are stored separately for each provider and tenant. The content of the credentials Secrets may vary for different use cases. The Secrets are labeled with the **hyperscalerType** and **tenantName** labels to manage pools of credentials for use by the provisioning process. This way, the in-use credentials and unassigned credentials available for use are tracked. Only the **hyperscalerType** label is added during Secret creation, and the **tenantName** label is added when the account corresponding to a given Secret is claimed. The contents of the Secrets are opaque to HAP.

The Secrets are stored in a Gardener seed cluster that HAP points to. They are available within a given Gardener project specified in the KEB and KIM configuration. This configuration uses a kubeconfig that gives KEB and KIM access to a specific Gardener seed cluster, which in turn enables access to those Secrets.

This diagram shows the HAP workflow:

![hap-workflow](../assets/hap-flow.drawio.svg)

Before a new cluster is provisioned, KEB queries for a Secret based on the **tenantName** and **hyperscalerType** labels.
If a Secret is found, KEB uses the credentials stored in this Secret. If a matching Secret is not found, KEB queries again for an unassigned Secret for a given hyperscaler and adds the **tenantName** label to claim the account and use the credentials for provisioning.

One tenant can use only one account per hyperscaler type.

This is an example of a Kubernetes Secret that stores hyperscaler credentials:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: {SECRET_NAME}
  labels:
    # tenantName is omitted for new, not yet claimed account credentials
    tenantName: {TENANT_NAME}
    hyperscalerType: {HYPERSCALER_TYPE}
```

## Shared Credentials

For a certain type of SAP BTP, Kyma runtimes, KEB can use the same credentials for multiple tenants.
In such a case, the Secret with credentials must be labeled differently by adding the **shared** label set to `true`. Shared credentials are not assigned to any tenant.
Multiple tenants can share the Secret with credentials. That is, many shoots (Shoot resources) can refer to the same Secret. This reference is represented by the SecretBinding (CredentialsBinding) resource.
When KEB queries for a Secret for a given hyperscaler, the least used Secret is chosen.  

This is an example of a Kubernetes Secret that stores shared credentials:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: {SECRET_NAME}
  labels:
    hyperscalerType: {HYPERSCALER_TYPE}
    shared: "true"
```

### Shared Credentials for the `sap-converged-cloud` Plan

For the `sap-converged-cloud` plan, each region is treated as a separate hyperscaler. Hence, Secrets are labeled with `openstack_{region name}`, for example, `openstack_eu-de-1`.

## EU Access

The [EU access](https://github.com/kyma-project/kyma-environment-broker/blob/main/docs/contributor/03-20-eu-access.md) regions require a separate credentials pool. The Secret contains the additional label **euAccess** set to `true`. This is an example of a Secret that stores EU access hyperscaler credentials:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: {SECRET_NAME}
  labels:
    # tenantName is omitted for new, not yet claimed account credentials
    tenantName: {TENANT_NAME}
    hyperscalerType: {HYPERSCALER_TYPE}
    euAccess: "true"
```

## Assured Workloads

SAP BTP, Kyma runtime supports the BTP `cf-sa30` Google Cloud subaccount region. This region uses the Assured Workloads Kingdom of Saudi Arabia (KSA) control package. Kyma Control Plane manages cf-sa30 Kyma runtimes in a separate
Google Cloud hyperscaler account pool. The Secret contains the label **hyperscalerType** set to `gcp_cf-sa30`. The following is an example of a Secret that uses the Assured Workloads KSA control package:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: {SECRET_NAME}
  labels:
    # tenantName is omitted for new, not yet claimed account credentials
    tenantName: {TENANT_NAME}
    hyperscalerType: "gcp_cf-sa30"
```

## Multi-Hyperscaler Accounts per Global Account

By default, one tenant can use only one hyperscaler account per provider. This limits the total number of clusters to the capacity of that single account.

To remove this restriction, KEB supports assigning multiple hyperscaler accounts to a tenant. Once a per-account cluster limit is reached, KEB automatically claims an additional hyperscaler account. Existing HAP account selection rules remain unchanged.

### Enable the Feature

Control the feature using the **allowedGlobalAccounts** configuration field:

| Configuration | Meaning |
|---|---|
| `allowedGlobalAccounts: []` or no config | Feature disabled for all global accounts |
| `allowedGlobalAccounts: ["GA-1", "GA-2"]` | Feature enabled only for the listed global accounts |
| `allowedGlobalAccounts: ["*"]` | Feature enabled for all global accounts |

### Cluster Limits per Hyperscaler Account

Each provider type has a configurable maximum number of clusters per account. When a CredentialsBinding reaches its limit, KEB provisions new clusters using a different account. The `default` limit applies to any provider that is not explicitly configured.

When a tenant has multiple accounts with available capacity, KEB uses the **fill-most-populated** strategy. It selects the account with the most clusters that is still below the limit. This maximizes the chance that the least-used account can be fully drained and reclaimed by the cleanup job over time.

### Provisioning Flow

When the feature is disabled (global account not in **allowedGlobalAccounts**), the following actions take place:

1. HAP rules determine the labels used to select the CredentialsBinding.
2. KEB queries for a CredentialsBinding matching the defined labels.
3. If none is found, KEB claims a new CredentialsBinding.
4. KEB provisions the cluster using the selected CredentialsBinding.

When the feature is enabled (global account is in **allowedGlobalAccounts**), the following actions take place:

1. HAP rules determine the labels used to select the CredentialsBinding.
2. KEB queries for all CredentialsBindings matching the defined labels.
3. If none are found, KEB claims a new CredentialsBinding.
4. If accounts are found, KEB queries the database for cluster counts and selects the most-populated account still below the limit.
5. If all accounts are at the limit, KEB claims a new CredentialsBinding.
6. KEB provisions the cluster using the selected CredentialsBinding.

The following is an example with an AWS limit of 180:

| Situation | Action |
|---|---|
| 150 clusters on CredentialsBinding-A | KEB provisions on CredentialsBinding-A, which is below the limit. |
| 180 clusters on CredentialsBinding-A | KEB claims CredentialsBinding-B and provisions the cluster on it. |
| 179 on A, 150 on B | KEB provisions on A, which is still below the limit, using the fill-most-populated strategy. |
| 180 on A, 150 on B | KEB provisions on B using the fill-most-populated strategy, because A has reached its limit. |

Accounts that already exceed the configured limit continue to work. KEB routes new clusters to a different account, and existing clusters on the over-limit account continue to work normally. Once the cluster count on the over-limit account drops below the configured limit, it becomes eligible for new clusters again.

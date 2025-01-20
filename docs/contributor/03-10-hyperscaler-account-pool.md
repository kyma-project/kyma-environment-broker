# Hyperscaler Account Pool

To provision clusters through Gardener using Runtime Provisioner, Kyma Environment Broker (KEB) requires a hyperscaler (GCP, Azure, AWS, etc.) account/subscription. Managing the available hyperscaler accounts is not in the scope of KEB. Instead, the available accounts are handled by Hyperscaler Account Pool (HAP).

HAP stores credentials for the hyperscaler accounts that have been set up in advance in Kubernetes Secrets. The credentials are stored separately for each provider and tenant. The content of the credentials Secrets may vary for different use cases. The Secrets are labeled with the **hyperscaler-type**, **euAccess**, **shared** and **tenant-name** labels to manage pools of credentials for use by the provisioning process. The **hyperscaler-type** contains hyperscaler name and region information in the format `hyperscaler_type: <HYPERSCALER_NAME>[_<PLATFORM_REGION>][_<HYPERSCALER_REGION>]`, where both `_<PLATFORM_REGION>` and `_<HYPERSCALER_REGION>` are optional. The **euAccess** and **shared** labels contain boolean values and they used to divide existing pools to secrets used by EU restricted regions and secrets shared by multiple Global Accounts. The **tenant-name** label is added when the account respective for a given Secret is claimed and it is the only one not added during Secret creation. This way, the in-use credentials and unassigned credentials available for use are tracked. The content of the Secrets is opaque to HAP.

The Secrets are stored in a Gardener seed cluster pointed to by HAP. They are available within a given Gardener project specified in the KEB and Runtime Provisioner configuration. This configuration uses a `kubeconfig` that gives KEB and Runtime Provisioner access to a specific Gardener seed cluster, which, in turn, enables access to those Secrets.

This diagram shows the HAP workflow:

![hap-workflow](../assets/hap-flow.drawio.svg)

Before a new cluster is provisioned, KEB queries for a Secret based on the mandatory **hyperscaler-type** and optional **tenant-name**, **euAccess** and **shared** labels. The query depeneds on plan/region rules configuration based of which KEB constructs secret bindings filter.

If a Secret is found, KEB uses the credentials stored in this Secret. If a matching Secret is not found, KEB queries again for an unassigned Secret for a given hyperscaler and adds the **tenant-name** label to claim the account and use the credentials for provisioning.

One tenant can use only one account per given hyperscaler type.

This is an example of a Kubernetes Secret that stores hyperscaler credentials:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: {SECRET_NAME}
  labels:
    # tenant-name is omitted for new, not yet claimed account credentials
    tenant-name: {TENANT_NAME}
    hyperscaler-type: {HYPERSCALER_TYPE}
```

## Shared Credentials

For a certain type of SAP BTP, Kyma runtimes, KEB can use the same credentials for multiple tenants.
In such a case, the Secret with credentials must be labeled differently by adding the **shared** label set to `true`. Shared credentials will not be assigned to any tenant.
Multiple tenants can share the Secret with credentials. That is, many shoots (Shoot resources) can refer to the same Secret. This reference is represented by the SecretBinding resource.
When KEB queries for a Secret for the given hyperscaler, the least used Secret is chosen.  

This is an example of a Kubernetes Secret that stores shared credentials:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: {SECRET_NAME}
  labels:
    hyperscaler-type: {HYPERSCALER_TYPE}
    shared: "true"
```

### Shared Credentials for `sap-converged-cloud` Plan

For the `sap-converged-cloud` plan, each region is treated as a separate hyperscaler. Hence, Secrets are labeled with **openstack_{region name}**, for example, **openstack_eu-de-1**.

## EU Access

The [EU access](03-20-eu-access.md) regions need a separate credentials pool. The Secret contains the additional label **euAccess** set to `true`. This is an example of a Secret that stores EU access hyperscaler credentials:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: {SECRET_NAME}
  labels:
    # tenant-name is omitted for new, not yet claimed account credentials
    tenant-name: {TENANT_NAME}
    hyperscaler-type: {HYPERSCALER_TYPE}
    euAccess: "true"
```

## Assured Workloads

SAP BTP, Kyma runtime supports the BTP cf-sa30 GCP subaccount region. This region uses the Assured Workloads Kingdom of Saudi Arabia (KSA) control package. Kyma Control Plane manages cf-sa30 Kyma runtimes in a separate
Google Cloud hyperscaler account pool. The Secret contains the label **hyperscaler-type** set to `gcp_cf-sa30`. The following is an example of a Secret that uses the Assured Workloads KSA control package:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: {SECRET_NAME}
  labels:
    # tenant-name is omitted for new, not yet claimed account credentials
    tenant-name: {TENANT_NAME}
    hyperscaler-type: "gcp_cf-sa30"
```

## Selection Rules

### Overview

HAP evaluates a set of rules to determine what labels to use when querying secret bindings. Input to the rules consists of an SKR's plan, hyperscaler and region. By default, only unchanged hyperscaler type value (values like aws, azure etc.) is used to search for the right pool.  If evaluated to true the rules modify or add labels used in the secret bindings resource query. There are four possible rules to configure:
* `hap.platformRegionRule` - if evaluated to true the `_<PLATFOR_REGION>` is appended to the `hyperscaler-type` label when searching, refered to as platform region based search. 
* `hap.clusterRegionRule` - if evaluated to true the `_<HYPERSCALER_REGION>` is appended to the `hyperscaler-type` label when searching, refered to as cluster region based search.
* `hap.sharedRule` - if evaluated to true the `shared: true` label is used when searching, refered to as shared based search.
* `hap.euAccessRule` - if evaluated to true the `shared: true` label is used when searching, refered to as euAccess based search.
The configuration is done by specifying above rules in KEB helm values.

Rules are independent of each other. Each plan and region combination can occur in the all rules specified above at the same time.

### Format

Each rule consists of a semicolon separated list of plans that the rule applies to. Additionally, a plan can be extended with `:region` suffix that makes the rule evaluate to true only if a cluster is provisioned in the specified region. List entries comply with the format `<PLAN_ID_1>:<REGION_ID_1>;<PLAN_ID_2>:<REGION_ID_2>`. Either plan or region (but never both) can be specified as wildcard `*` meaning all plans or regions should apply for specific second value. The following example lists valid and invalid configuration values:
* `trial` - valid, rule is evaluated to true for the `trial` plan in all regions,
* `trial:*` - valid, rule is evaluated to true for the `trial` plan in all regions,
* `trial:eu` - valid, rule is evaluated to true for the `trial` plan in the `eu` region,

* `*:eu` - valid, rule is evaluated to true for all plans in the `eu` region,
* `eu:*` - invalid, plan must be specified in the first part of `<PLAN_ID>:<REGION_ID>` pair,
* `*:*` - invalid, at least one of the values must be specified in the `<PLAN_ID>:<REGION_ID>` pair.
* `*:eu;trial:eu` - valid, rule is evaluated to true for all plans in the `eu` region and for the `trial` plan in the `eu` region, configuration can be duplicated
* `trial:eu;trial:gcp` - valid, rule is evaluated to true for trials plans but only in `eu` and `gcp` regions,

### Validation

Rules validation is done during the KEB startup. If the configuration is invalid, KEB will not start and an error message will be displayed in the logs. The constraints used for validation include:
* Rules format check - all the rules must comply with the format specified above.
* Plan existence check.

### Examples

The section provides configuration examples of secret bindings selection rules and shows what hyperscaler pools they correspond to.

By default, KEB uses the `hyperscaler-type` label to search for secrets without applying any sufixes to the label's value. It means that without any configuration the search is done by simple hyperscaler type value implemented in [KEB](https://github.com/kyma-project/kyma-environment-broker/blob/main/common/hyperscaler/hyperscaler_type.go).

```
hap: 
  platformRegionRule: ""
  clusterRegionRule: ""
  sharedRule: ""
  euAccessRule: ""
```

![hap-workflow](../assets/default-pool.drawio.svg)

---

The following configuration is an example of a rule that appends the platform region to the `hyperscaler-type` label when searching for secrets for all gcp clusters. The configuration means that there would be no possibillity to configure a pool with label `hyperscalerType: gcp` since KEB would search for `hyperscalerType: gcp__<PLATFOR_REGION>` bindings only.

```
hap: 
  platformRegionRule: "gcp"
  clusterRegionRule: ""
  sharedRule: ""
  euAccessRule: ""
```

![hap-workflow](../assets/gcp-platform-pool.drawio.svg)

---

It is possible to extend the configuration rule entry with region specified. Below configuration means that if a cluster is provisioned in cf-sa30 subaccount regiont then KEB would search for secret bindings with `hyperscalerType: gcp_cf-sa30` label. In other cases, for example when probisioning a gcp cluster in other region, KEB would search for `hyperscalerType: gcp` label. Differently than in previous example, in this case, it is possible to configure two different pools for the same hyperscaler type.

```
hap: 
  platformRegionRule: "gcp:cf-sa30"
  clusterRegionRule: ""
  sharedRule: ""
  euAccessRule: ""
```

![hap-workflow](../assets/gcp-platform-region-pool.drawio.svg)

---

As stated previously, it is possible to include multiple elements in each configuration rule. The example below shows how to configure multiple plan/region pairs for based on platformRegionRule. The same can be configured for all other rules (clusterRegionRule, sharedRule, euAccessRule). The list below allows to KEB to operate on 3 pools for gcp plan: one withouth a region, one for cf-sa30 region and one for jp30 region; 2 pools for aws plan: one without a region and one for eu11 region; multiple pools for openstack plan - all with a platform region specified; and one pool for azure without a platform region.

```
hap: 
  platformRegionRule: "gcp:cf-sa30;gcp:cf-jp30;aws:cf-eu11;openstack"
  clusterRegionRule: ""
  sharedRule: ""
  euAccessRule: ""
```

![hap-workflow](../assets/gcp-platform-region-pool-list.drawio.svg)

---

Below configuration specifies azure plan as one for which to use cluster region based search. In this case, cluster provisioned in with the azure plan would require a secret binding with `hyperscalerType: azure_<CLUSTER_REGION>` label. All of the rule examples apply all of platformRegionRule, clusterRegionRule, shareRule and euAccessRule properties.

```
hap: 
  platformRegionRule: "gcp"
  clusterRegionRule: "azure"
  sharedRule: ""
  euAccessRule: ""
```

![hap-workflow](../assets/gcp-azure-pool.drawio.svg)

---

All the configuration rules are independent of each other from the point of view of their configuration. If the same plan/region pair appears in more than one property, then all the rules that it appears in take effect. Below configuration specifies that for all gcp clusters the search should be done based on platform region and cluster region searches. The resulting search label would be `hyperscalerType: gcp_<PLATFORM_REGION>_<CLUSTER_REGION>`.

```
hap: 
  platformRegionRule: "gcp"
  clusterRegionRule: "gcp"
  sharedRule: ""
  euAccessRule: ""
```

![hap-workflow](../assets/gcp-cluster-pool.drawio.svg)

---

`sharedRule` and `euAccessRule` are idependent but in contrast to `platformRegionRule` and `clusterRegionRule` if they are both evaluated to true the search label contains two additional labels `shared: true` and `euAccess: true`. Instead of modifying the same label value `hyperscalerType` two additional labels are added to the same query. In the example below, all gcp clusters will use the same pool of secret bindings marked with labels: `hyperscalerType: gcp_<PLATFORM_REGION>_<CLUSTER_REGION>`, `shared: true`, `euAccess: true`.

```
hap: 
  platformRegionRule: "gcp"
  clusterRegionRule: "gcp"
  sharedRule: "gcp"
  euAccessRule: "gcp"
```

![hap-workflow](../assets/all-gcp-pool.drawio.svg)

---

The last example shows initial configuration create to mimic the current bahaviour of KEB at the time of writing the document. The configuration enforces that:
* azure, aws, gcp have their own pools of dedicated bindings.
* gcp clusters in the region cf-sa30 use the pool of secret bindings marked with labels: `hyperscalerType: gcp_cf-sa30`,
* sap-converged-cloud clusters use the pool of secret bindings marked with labels: `hyperscalerType: openstack_<CLUSTER_REGION>` and all of the are shared.
* trial clusters can use one of two pool of shared secret bindings marked with labels: `hyperscalerType: azure` or `hyperscalerType: aws` depending on the provider type.
* azure clusters in the region cf-ch20 and aws clusters in the region cf-eu11 have their own dedicated pool.
```
hap: 
  platformRegionRule: "gcp:cf-sa30"
  clusterRegionRule: "sap-converged-cloud"
  sharedRule: "trial;sap-converged-cloud"
  euAccessRule: "azure:cf-ch20;aws:cf-eu11"
```

![hap-workflow](../assets/initial-pools.drawio.svg)

---
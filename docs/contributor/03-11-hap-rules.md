# HAP Rule Configuration

<!--

TODO: The below is snippet from the old docu. Think where to put it.

## Examples

The section provides configuration examples of secret bindings selection rules and shows what hyperscaler pools they correspond to.

By default, KEB uses the `hyperscaler-type` label to search for secret bindings without applying any sufixes to the label's value. It means that without any configuration the search is done by simple hyperscaler type value implemented in [KEB](https://github.com/kyma-project/kyma-environment-broker/blob/main/common/hyperscaler/hyperscaler_type.go). -->

- Hap rule configuration is a functionality that allows to extract hardcoded pool selection instructions to a configurable helm property.
- The pool is a set of SecretBindings that share the sape search labels
- The pool selection is a process that for every skr creation request decides on which secret binding is the most suitable to use by generating a list of search labels based on skr's physical and logical attributes.
- the search labels are used to query Gardener's Secret Bindings
- more about surrounding process can be found in HAP doc (TODO: link)
- The actual decision is made by evaluating a rule and if it applies to the request then it applies its own search labels extensions in to the query of a SecretBinding

![Rules Evaluation Process](../assets/hap-rules-general.drawio.svg)

## Rule Configuration

- pools are configured in values.yaml in `hap.rule` property
- `hap.rule` is a list of strings, each of which is a single entry the corresponds to a single hyperscaler pool
- each entry in the list is a rule that is evaluated during the cluster provisioning process
- if more than one rule applies the best matching rule is selected based on the priority, which is described in the Rule Priority section

```
hap: 
  rule: 
  - rule_1
  - rule_2
  ...
  - rule_n
```
- every pool that the rule represents must be preconfigured

## Rule Format 

- PLAN(ATTR_1=VAL_1,ATTR_2=VAL_2,...,ATTR_N)
- In minimal form each rule consists of a plan that the rule applies to. 
- The rule can be extended with a list of attributes and its values
- list of possible attributes is described in the following sections
- description of when the rule is triggered can be found in the following sections

## Evaluation

<!-- rules input -->
HAP evaluates a set of rules to determine what labels to use when querying secret bindings. Input to the rules consists of an SKR's plan, hyperscalerRegion and clusterRegion.

- rules are evaluated during cluster provisioning
- rule entries are analyzed one by one
- plan and attributes from every entry are compared with values from input, if they are the same the rule entry is considered triggered - in order for the rule to be triggered its plan and attributes must match what is passed in the request
- if there is more than one triggered rule then only one is selected in accordance to priority rules
- if there are no rule entries trigered then the error is returned, there is no default secret
- triggered rule entry applies its specific labels to the SecretBinding search query

## Priority

- only one rule can be triggered
- if more than one rule entry match the request than only one is selected by sorting them by the number of attributes they contain  
- the entry that specifies most attributes (hence is more specific) is selected
- if there are more than one rule with the same number of attributes then an error is returned

## Search Labels

<!-- default search -->
- HAP stores credentials for the hyperscaler accounts that have been set up in advance in Kubernetes Secrets. The credentials are stored separately for each provider and tenant.
- By default, **hyperscalerType** label value without any suffix (values like aws, azure etc.) is used to search for the right pool.
- If evaluated to true the rules modify or add labels used in the secret bindings resource query.
- The output labels include: **hyperscalerType**, **shared**, **euAccess**. The **hyperscaler-type** contains hyperscaler name and region information in the format `hyperscaler_type: <HYPERSCALER_NAME>[_<PLATFORM_REGION>][_<HYPERSCALER_REGION>]`, where both `_<PLATFORM_REGION>` and `_<HYPERSCALER_REGION>` are optional. The **euAccess** and **shared** labels contain boolean values and they used to divide existing pools to secrets used by EU restricted regions and secrets shared by multiple Global Accounts. 
- **hyperscaler-type** is mandatory
- **euAccess** and **shared** are optional

## Validation

- validated during startup
- during startup the rull will look at if correct pools are defined

Rules validation is done during the KEB startup. If the configuration is invalid, KEB will not start and an error message will be displayed in the logs. The constraints used for validation include:
* Rules format check - all the rules must comply with the format specified above.
* Plan existence check.

## Empty Pool

Describe:
- Result: error, no rule configured

```
hap: 
  rule: ""
```

## Simple Rules

```
hap: 
  rule: 
  - azure
  - sap-converged-cloud

SecretBinding pools
- hyperscalerType: trial, 
- hyperscalerType: openstack, 
```

Described:
- you defined rules by listing plans 
- plan will be translated to search labels
- in the basic form rule contains plan but is translated to hyperscalerType
- extended expresions alter the search query by addition different properties

## Rule attributes

- physical vs logical

Existence of an attribute in the rule means its inclusion in the search label. If a have a triggered rule without attributes than this attribute is not included in the query.

- attributes: plan, platformRegion, clusterRegion, shared, euAccess

### Platform Region Attribute

It is possible to extend the configuration rule entry with specified region. Below configuration means that if a cluster is provisioned in cf-sa30 subaccount region then KEB would search for secret bindings with `hyperscalerType: gcp_cf-sa30` label. In other cases, for example when probisioning a gcp cluster in other region, KEB would search for `hyperscalerType: gcp` label. Differently than in previous example, in this case, it is possible to configure two different pools for the same hyperscaler type.

```
hap: 
  rule: 
    - azure
    - aws
    - gcp
    - gcp(PR=cf-sa30)
    - sap-converged-cloud

SecretBinding pools
- hyperscalerType: azure, 
- hyperscalerType: aws, 
- hyperscalerType: gcp, 
- hyperscalerType: gcp_cf-sa30, 
- hyperscalerType: openstack, 
```

### Cluster Region Attribute

Below configuration specifies azure plan as one for which to use cluster region based search. In this case, cluster provisioned in with the azure plan would require a secret binding with `hyperscalerType: azure_<CLUSTER_REGION>` label. All of the rule examples apply all of platformRegionRule, clusterRegionRule, shareRule and euAccessRule properties.

```
hap: 
  rule:
    - gcp(PR=*)
    - azure(CR=*)
    - aws
    - sap-converged-cloud 

TODO: make the section based on the specific region instead of an *

SecretBinding pools
- hyperscalerType: azure_<CLUSTER_REGION>, 
- hyperscalerType: aws, 
- hyperscalerType: gcp_<PLATFOR_REGION>, 
- hyperscalerType: openstack, 
```


All the configuration rules are independent of each other from the point of view of their configuration. If the same plan/region pair appears in more than one property, then all the rules that it appears in take effect. Below configuration specifies that for all gcp clusters the search should be done based on platform region and cluster region searches. The resulting search label would be `hyperscalerType: gcp_<PLATFORM_REGION>_<CLUSTER_REGION>`.

### Shared attribute

`sharedRule` and `euAccessRule` are idependent but in contrast to `platformRegionRule` and `clusterRegionRule` if they are both evaluated to true the search label contains two additional labels `shared: true` and `euAccess: true`. Instead of modifying the same label value `hyperscalerType` two additional labels are added to the same query. In the example below, all gcp clusters will use the same pool of secret bindings marked with labels: `hyperscalerType: gcp_<PLATFORM_REGION>_<CLUSTER_REGION>`, `shared: true`, `euAccess: true`.

```
hap: 
  rule:
  - azure
  - aws
  - gcp(CR=*, PR=*, euAccess=true, shared=true)
  - sap-converged-cloud

SecretBinding pools:
- hyperscalerType: azure, 
- hyperscalerType: aws, 
- hyperscalerType: gcp, 
- hyperscalerType: gcp_<PLATFORM_REGION>_<CLUSTER_REGION>, 
...
- hyperscalerType: openstack, 
```

TODO: region/platform variations

### euAccess attribute


### Pools Divided per Attribute

As stated previously, it is possible to include multiple elements in each configuration rule. The example below shows how to configure multiple plan/region pairs for based on platformRegionRule. The same can be configured for all other rules (clusterRegionRule, sharedRule, euAccessRule). The list below allows to KEB to operate on 3 pools for gcp plan: one withouth a region, one for cf-sa30 region and one for cf_jp30 region; 2 pools for aws plan: one without a region and one for cf_eu11 region; multiple pools for openstack plan - all with a platform region specified; and one pool for azure without a platform region.

```
hap: 
  rule:
    - azure
    - aws
    - aws(PR=cf-eu11)
    - gcp
    - gcp(PR=cf-sa30)
    - gcp(PR=cf-jp30)
    - sap-converged-cloud

SecretBinding pools
- hyperscalerType: azure, 
- hyperscalerType: aws, 
- hyperscalerType: aws_cf-eu11, 
- hyperscalerType: gcp, 
- hyperscalerType: gcp_cf-sa30, 
- hyperscalerType: gcp_cf-jp30, 
- hyperscalerType: openstack, 
```



### Multiple Attributes in the same rule

```
hap: 
  rule:
  - gcp(CR=*, PR=*)
  - azure
  - aws
  - sap-converged-cloud

SecretBinding pools:
- hyperscalerType: azure
- hyperscalerType: aws
- hyperscalerType: gcp_<PLATFORM_REGION>_<CLUSTER_REGION>
- hyperscalerType: openstack
```




### Attribute with "*"

The following configuration is an example of a rule that appends the platform region to the `hyperscaler-type` label when searching for secrets for all gcp clusters. The configuration means that there would be no possibillity to configure a pool with label `hyperscalerType: gcp` since KEB would search for `hyperscalerType: gcp_<PLATFORM_REGION>` bindings only.

```
hap: 
  rule: 
  - aws(PR=*)

SecretBinding pools
- hyperscalerType: aws_PLATFOR_REGION, 
```

TODO: note that we have removed gcp pool at all

TODO: An example should explain rules priority, how plan translates to hyperscaler account type (sap-converged-cloud -> openstack) and what asterix means

## Initial Configuration

The last example shows initial configuration create to mimic the current bahaviour of KEB at the time of writing the document. The configuration enforces that:
* azure, aws, gcp have their own pools of dedicated bindings.
* gcp clusters in the region cf-sa30 use the pool of secret bindings marked with labels: `hyperscalerType: gcp_cf-sa30`,
* sap-converged-cloud clusters use the pool of secret bindings marked with labels: `hyperscalerType: openstack_<CLUSTER_REGION>` and all of the are shared.
* trial clusters can use one of two pool of shared secret bindings marked with labels: `hyperscalerType: azure` or `hyperscalerType: aws` (because of hardcoded mapping of trial plan to azure or aws providers) depending on the provider type.
* azure clusters in the region cf-ch20 and aws clusters in the region cf-eu11 have their own dedicated pool.

```
hap: 
  rule:
  - azure(euAccess=*)
  - aws(euAccess=*)
  - trial(shared)
  - gcp
  - gcp(PR=cf-sa30) 
  - openstack(CR=*, shared)

  platformRegionRule: "gcp:cf-sa30"
  clusterRegionRule: "sap-converged-cloud"
  sharedRule: "trial;sap-converged-cloud"
  euAccessRule: "azure:cf-ch20;aws:cf-eu11"

TODO: rules are not restrictive - definition of a rule with "*" does not mean that all variations of pools must be created, this is the only place where validation does not take place
TODO: translation between plan and hyperscaler type

SecretBinding pools:
- hyperscalerType: azure, 
- hyperscalerType: aws, 
- hyperscalerType: gcp, 
- hyperscalerType: azure; shared: true - TRIAL POOL
- hyperscalerType: aws; shared: true - TRIAL POOL 
- hyperscalerType: azure; euAccess: true 
- hyperscalerType: aws; euAccess: true 
- hyperscalerType: gcp_cf-sa30, 
- hyperscalerType: openstack_<CLUSTER_REGION>; shared: true, 
```
---
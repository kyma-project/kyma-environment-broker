# HAP Rule Configuration

> [!NOTE]
> This feature is still being developed and will be available soon.

The Hyperscaler Account Pool (HAP) rule configuration is a functionality that allows you to control pool selection from a configurable helm property. 
A pool is a set of Gardener's SecretBinding resources that share the same search labels. 
Pool selection is a process that decides which SecretBinding pool is the most suitable for every SAP BTP, Kyma runtime creation request.
The process generates a list of [search labels](#search-labels) based on Kyma runtime's attributes.

The search labels are used to query Gardener's secret bindings.

![Rules Evaluation Process](../assets/hap-rules-general.drawio.svg)

## Rule Configuration

The rule is configured in `values.yaml` in the `hap.rule` property.
It is a list of strings, where each string is a single rule entry that corresponds to a single pool.
Each rule entry is evaluated during cluster provisioning.
If more than one rule applies, the best matching rule is selected based on the [priority](#uniqueness-and-priority). See an example configuration in `values.yaml`:

```
hap: 
  rule: 
  - rule_entry_1
  - rule_entry_2
  ...
  - rule_entry_n
```

Every pool represented by a rule entry must be preconfigured in a Gardener cluster that Kyma Environment Broker (KEB) connects to.

## Rule Format 

See the following example of a rule entry format:

```
PLAN(INPUT_ATTR_1=VAL_1, INPUT_ATTR_2=VAL_2, ..., INPUT_ATTR_N=VAL_N) -> OUTPUT_ATTR_1, OUTPUT_ATTR_2, ..., OUTPUT_ATTR_M
```

Every rule entry consists of input and output attributes separated by the arrow symbol - `->`.
The input attributes match a rule entry with the Kyma runtime request and modify [search labels](#search-labels). The output attributes only modify the search labels.
In its minimal form, each rule consists of a **PLAN** that the rule applies to.
In its extended form, a rule entry contains lists of input attributes. Their values are passed as `ATTR=VAL` pairs in parentheses.

> [!NOTE]
> If you do not provide any values, use empty parentheses or do not use them at all, for example:
> * `aws`
> * `aws()`

Output attributes do not support values.
To learn about rule attributes and when the rule is triggered, see the [Rule Evaluation](#rule-evaluation) section.

The possible **OUTPUT_ATTR_x** attribute values are `S` or `EU`.

```
hap: 
  rule: 
  - azure(INPUT_ATTR_1=VAL_1,INPUT_ATTR_2=VAL_2,...,INPUT_ATTR_N) -> OUTPUT_ATTR_1=VAL_1,OUTPUT_ATTR_2=VAL_2,...,OUTPUT_ATTR_M
```

The input attribute types include **platformRegion** (**PR**) and **hyperscalerRegion** (**HR**). The output attribute types include **shared** (**S**) and **euAccess** (**EU**). 
You can only use each attribute once in a single rule entry.
Use the **shared** and **euAccess** attributes only to apply their labels if the other matched attributes are equal to Kyma runtime's values. 

## Rule Evaluation

During cluster provisioning, HAP evaluates a set of rules to determine which labels to use when querying Secret bindings.
Rule entries are analyzed one by one.
Input attributes from every entry are compared with values from the Kyma runtime input. If they are the same, the rule entry is considered matched.
If more than one rule is triggered, only one is selected, as described in the [Priority](#uniqueness-and-priority) section.
If no rule entries are triggered, an error is returned. In this case, no fallback behavior is defined.

## Search Labels

Search labels are generated by evaluating a rule consisting of rule entries. If an entry applies to the request, the rule outputs its specific search labels to query the correct pool.
For more information on the HAP process, see [Hyperscaler Account Pool](03-10-hyperscaler-account-pool.md).

HAP stores credentials to hyperscaler accounts in Kubernetes Secrets that SecretBindings point to. KEB searches for SecretBindings using labels **hyperscalerType**, **shared**, and **euAccess**. 

The hyperscaler type contains a hyperscaler name and region information as `hyperscalerType: <HYPERSCALER_NAME>[_<PLATFORM_REGION>][_<HYPERSCALER_REGION>]`, where both `_<PLATFORM_REGION>` and `_<HYPERSCALER_REGION>` are optional. The **hyperscalerType** label is mandatory. Its value is computed based on the plan and regions provided, and mapped in [hyperscaler_type.go](https://github.com/kyma-project/kyma-environment-broker/blob/main/common/hyperscaler/hyperscaler_type.go). Not all plans share their name with hyperscaler types, for example, the `sap-converged-cloud` plan has the `openstack` hyperscaler type and the `trial` plan can have either `azure` or `aws` depending on the configured provider type. 

In all the cases, `HYPERSCALER_NAME` refers to a provider type. The following table shows the plan-provider type mapping:

| Plan                	| Provider Type 	|
|---------------------	|-----------------|
| `azure`               | `azure`         |
| `azure_lite`          | `azure`         |
| `aws`                 | `aws`           |
| `free`                | `azure`, `aws`  |
| `gcp`                 | `gcp`           |
| `preview`             | `aws`           |
| `sap-converged-cloud` | `openstack`     |
| `trial`              	| `aws`           |

The **euAccess** and **shared** labels contain boolean values and are optional. They divide existing pools between Secrets used by EU-restricted regions and Secrets shared by multiple global accounts.

Every rule must contain at least a plan and apply the `hyperscalerType: <HYPERSCALER_NAME>` label. See an example of a simple rule entry configuration and a SecretBinding pool that this configuration corresponds to:

```
hap:
  rule:
    - gcp                             # pool: hyperscalerType: gcp
```

All the examples in the document show the structure of configuration and corresponding pools in the comment as the snippet above.

## Rule Attributes

If an attribute exists in a rule, it constrains a set of clusters it applies to in the same way as the plan. The following section describes the attributes that you can use in rule entries.

### Platform Region Attribute

To extend a rule entry with a specified platform region, add the **PR** rule attribute. If the value of Kyma runtime's platform region attribute matches the **PR** rule attribute, the `hyperscalerType: <HYPERSCALER_NAME>_<PLATFORM_REGION>` label is used.

The following configuration means that if a `gcp` cluster is provisioned in the cf-sa30 platform region, KEB searches for secret bindings with the `hyperscalerType: gcp_cf-sa30` label. 

```
hap: 
  rule: 
    - gcp(PR=cf-sa30)                 # pool: hyperscalerType: gcp_cf-sa30
```

### Hyperscaler Region Attribute

A region where a Kyma runtime is provisioned can be matched with the **HR** attribute. The following configuration specifies the `azure` plan for using the hyperscaler region-based search. In this case, the cluster provisioned with the `azure` plan requires a secret binding with the `hyperscalerType: azure_<HYPERSCALER_REGION>` label. 

```
hap: 
  rule: 
    - gcp(HR=us-central1)             # pool: hyperscalerType: gcp_us-central1,
```

### Shared and EU Access Attributes

The **shared** and **euAccess** attributes do not correspond to any Kyma runtime property. Use these attributes only to add search labels. If the rule entry contains either of the attributes, then when the rule is triggered, the `shared: true` or `euAccess: true` labels are added to the [search labels](#search-labels). The shared label on a SecretBinding marks it as assignable to more than one Kyma runtime, and euAccess is used to mark EU regions (see [Hyperscaler Account Pool](03-10-hyperscaler-account-pool.md)). The following configuration specifies that all `gcp` clusters use the same pool of shared secret bindings marked with labels `hyperscalerType: gcp`, `shared: true`, and azure clusters in the region cf-ch20 use a pool of secret bindings marked with labels `hyperscalerType: azure`, and `euAccess: true`.

```
hap: 
  rule: 
    - gcp -> S                        # pool: hyperscalerType: gcp; shared: true 
    - azure(PR=cf-ch20) -> EU          # pool: hyperscalerType: azure_cf-ch20, euAccess: true
```

### Attribute with "*"

You can replace input attributes' values with `*`, which means that they match all Kyma runtime values of that attribute. They are used when creating [search labels](#search-labels). See the example configuration that makes KEB search for SecretBindings with `hyperscalerType: gcp_<PR>`, where **PR** matches the corresponding Kyma runtime value for all `gcp` clusters:

```
hap: 
  rule: 
  - gcp(PR=*)                         # pool: hyperscalerType: gcp_cf-sa30
                                      # pool: hyperscalerType: gcp_cf-jp30
```

The attributes that support `*` are: **PR**, **HR**.

> [!NOTE]
> This configuration example effectively disables the usage of SecretBindings labeled only with `hyperscalerType: gcp`.

### Attributes Summary

You can use the following attributes in the rule entry.



| Name (Symbol)        	| Data Type and Possible Values                                                                                                                    	| Input Attribute 	| Output Attribute 	| Modified SecretBinding Search Labels                                                              	|
|----------------------	|------------------------------------------------------------------------------------------------------------------------------------	|-----------------	|------------------	|---------------------------------------------------------------------------------------------------	|
| Platform Region (**PR**) 	| string, subaccount regions as defined in [Regions for the Kyma Environment](https://help.sap.com/docs/btp/sap-business-technology-platform/regions-for-kyma-environment) 	| true            	| false            	| hyperscalerType: `<providerType>_<PR>` or hyperscalerType: `<providerType>_<PR>_<HR>` if used with **HR** 	|
| Hyperscaler Region (**HR**)  	| string, cluster region as defined in [Regions for the Kyma Environment](https://help.sap.com/docs/btp/sap-business-technology-platform/regions-for-kyma-environment)   	| true            	| false            	| hyperscalerType: `<providerType>_<HR>` or hyperscalerType: `<providerType>_<PR>_<HR>` if used with **PR**	|
| EU Access (**EU**)       	| no value                                                                                                                           	| false           	| true             	| euAccess: true                                                                                    	|
| Shared (**S**)           	| no value                                                                                                                           	| false           	| true             	| shared: true                                                                                      	|

> [!NOTE] 
> * Input/Output attributes - true when the attribute can occur in the input/output attributes section (left part) of a rule entry (see [Rule Format](#rule-format) section).
> * Modified SecretBinding search labels - lists modifications of search labels that the rule entry containing the attribute adds to search labels if triggered. 

## Uniqueness and Priority

Only one rule can be triggered. If more than one rule entry matches the request, only one is selected and applied. The process of selecting the best matching rule is based on rule uniqueness and priority.
Rule entry uniqueness is determined by its plan and input parameters' values (identification attributes) not including input parameters with `*`. 
Output parameters are not taken into account when establishing rule entry uniqueness. For example, the following rule fails on startup because both rules can match every Kyma runtime:

```
hap:
  rule: 
    - gcp 
    - gcp -> S                      # invalid entry, output attributes do not take part into uniqueness check
    - gcp(PR=*)                     # invalid entry, both can be applied to the same Kyma Runtime
    - gcp(HR=europe-west3)          # valid entry, new HR attribute makes the rule unique
    - gcp(PR=*, HR=europe-west3)    # invalid rules, the same as previous one because of addition of `PR=*` attribute
```

Rule configuration must contain only unique entries.
Otherwise, an error that fails KEB's startup is returned.

Rule entry priority is selected by sorting all rule entries that apply to the request by the number of identification attributes they contain. 
For example, a rule including only a plan and no attributes has lower priority than a rule with the same plan and a platform region attribute (`gcp` < `gcp(PR=cf-sa30)`).
After sorting, the entry that specifies the most attributes is selected because it is the most specific. 
Input attributes with value `*` are not taken into account when calculating priority.

The following example shows the priority of the listed rules starting from the lowest:

```
aws -> S                              # search labels: hyperscalerType: aws, shared: true
aws(PR=cf-eu11) -> EU                 # search labels: hyperscalerType: aws_cf-eu11, euAccess: true
aws(PR=cf-eu11, HR=westeu) -> EU, S   # search labels: hyperscalerType: aws_cf-eu11_westeu, shared: true, euAccess: true
```

## Validation

KEB validates HAP rules during startup. If the configuration is invalid, KEB does not start, and an error message is displayed in the logs. 

The constraints used for validation during KEB startup include the following:
* Rules format check: All the rules must comply with the specified format.
* Every supported plan needs at least one rule entry; if no rule entry is defined for a plan,  an error is returned during KEB startup.
* Uniqueness validation check: KEB checks if all rule entries are unique in the rule's scope. You must not specify more than one entry with the same number of identification attributes. Otherwise, the error is returned, failing KEB's startup. For more details, see the [Uniqueness and Priority](#uniqueness-and-priority) section. 

## Initial Configuration

The following example shows the initial configuration created to mimic KEB behavior. The configuration enforces the following:
* The `azure`, `aws`, and `gcp` plans have their own pools of dedicated bindings.
* The `free` plan uses `aws` or `azure` dedicated bindings, depending on the provider value.
* `gcp` clusters in the cf-sa30 region use the pool of secret bindings marked with the `hyperscalerType: gcp_cf-sa30` label.
* `sap-converged-cloud` clusters use the pool of secret bindings marked with the `hyperscalerType: openstack_<HYPERSCALER_REGION>` label, and all these pools are shared.
* `trial` clusters can use one of two pools of shared secret bindings marked with labels `hyperscalerType: azure` or `hyperscalerType: aws`, depending on the `trial` provider type used.
* `azure` clusters in the cf-ch20 region and `aws` clusters in the cf-eu11 region have their own dedicated pools and are euAccess specific.

See an example of the initial configuration created to mimic KEB behavior.
```
hap:
 rule: 
  - aws                             # pool: hyperscalerType: aws
  - aws(PR=cf-eu11) -> EU           # pool: hyperscalerType: aws_cf-eu11; euAccess: true 
  - azure                           # pool: hyperscalerType: azure
  - azure(PR=cf-ch20) -> EU         # pool: hyperscalerType: azure_cf-ch20; euAccess: true 
  - gcp                             # pool: hyperscalerType: gcp
  - gcp(PR=cf-sa30)                 # pool: hyperscalerType: gcp_cf-sa30
  - trial -> S                      # pool: hyperscalerType: azure; shared: true - TRIAL POOL
                                    # pool: hyperscalerType: aws; shared: true - TRIAL POOL 
  - sap-converged-cloud(HR=*) -> S  # pool: hyperscalerType: openstack_<HYPERSCALER_REGION>; shared: true
  - azure_lite                      # pool: hyperscalerType: azure
  - preview                         # pool: hyperscalerType: aws
  - free                            # pool: hyperscalerType: aws
                                    # pool: hyperscalerType: azure
```

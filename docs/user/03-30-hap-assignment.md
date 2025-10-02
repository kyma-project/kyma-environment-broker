# Assigning Kyma Instances to IaaS Provider Accounts

Each IaaS provider (Amazon Web Services, Google Cloud, Microsoft Azure, SAP Cloud Infrastructure) provides pools of hyperscaler accounts <!--or just accounts? must check later--> to which your Kyma instances are assigned. The assignment is primarily based on the IaaS provider type. This means that Kyma instances in one global account <!--represented by the same global account ID--> are assigned to a single account of a particular IaaS provider. The IaaS provider corresponds to a Kyma instance's plan and, in some cases, other requirements, such as EU Access. For example, a Kyma instance with the standard `aws` plan is assigned to an Amazon Web Services account, and a Kyma instance with a standard `build-runtime-azure` plan is assigned to a Microsoft Azure account.

## IaaS Provider Account Assignment Rules

The specific assignment options are the following:

- Amazon Web Services offers the following account pools:
  - An account pool for Kyma instances created with the standard service plans `aws`, `build-runtime-aws`, and the `free` plan
  - An account pool for Kyma instances created with the standard service plans `aws` and `build-runtime-aws` in the subaccount <!--platform?--> region cf-eu11, Europe (Frankfurt) EU Access
  - A pool of shared accounts for trial Kyma instances

- Microsoft Azure offers the following account pools:
  - An account pool for Kyma instances created with the standard service plans `azure`, `build-runtime-azure`, test demo and development plan `azure-lite`, and the `free` plan
  - An account pool for Kyma instances created with the standard service plans `azure`, `build-runtime-azure` in the subaccount <!--platform?--> region cf-ch20, Switzerland (Zurich) EU Access

- Google Cloud offers the following account pools:
  - An account pool for Kyma instances created with the standard service plans `gcp` and `build-runtime-gcp`
  - An account pool for Kyma instances created with the standard service plans `gcp` and `build-runtime-gcp` in the subaccount <!--platform?--> region cf-sa30, KSA (Dammam) GCP public sector, requiring Assured Workloads Kingdom of Saudi Arabia (KSA) control package.

- <!--INTERNAL! Check this!!!!--> SAP Cloud Infrastructure offers one pool of shared accounts for each of the cluster regions available for the service plan `sap-converged-cloud`


For example, once you create the first Kyma instance with the standard `aws` plan in a non-EU region in your global account, it is automatically assigned to an account provided by Amazon Web Services. After that, all your Kyma instances in all subaccounts within the same global account created with the `aws` or `build-runtime-aws` plans in non-EU regions are assigned to the same Amazon Web Services account. Suppose you have no Kyma instances created in the `aws` EU region in the same global account. In that case, this is your only assignment to an Amazon Web Services account, regardless of the number of existing Kyma instances with the standard `aws` plan. However, if, in the same global account, there are also Kyma instances created with the `aws` or `build-runtime-aws` plans in the `aws` EU region, they are all assigned to another single account provided by Amazon Web Services. Therefore, your Kyma instances in this particular global account are assigned to two IaaS provider accounts. Adding more Kyma instances with other plans results in more assignments to the corresponding IaaS provider accounts.

## Migrating Assigned Kyma Instances

The assignment is permanent and cannot be changed. This is also true when you migrate subaccounts with Kyma instances assigned to a specific IaaS provider account from one global account to another. The original account assignments remain unchanged even if the migrated Kyma instances and those existing in the target global account have the same characteristics.

For example, if you migrate Kyma instances created with the standard `aws` service plan, all assigned to a single account provided by Amazon Web Services, to another global account with existing Kyma instances also created with the standard `aws` service plan, all assigned to another account provided by Amazon Web Services, the assignments remain unchanged. Even though all your Kyma instances in this particular global account share the same characteristics, they remain assigned to two different IaaS provider accounts.
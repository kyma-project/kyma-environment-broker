# Multi-Hyperscaler Accounts per Global Account

## Status

Accepted

## Context

For regular provisioning, a global account (GA) can use only one hyperscaler account per provider, limiting cluster count to the account's capacity.

> ### Note: 
> Multiple accounts per global account are already supported for special cases (for example, AWS and EU Access AWS cluster in the same GA). However, automatic assignment based on capacity limits is not available.

The Hyperscaler Account Pool (HAP) selects a CredentialsBinding by matching the label `tenantName={GA}`. If none exists, a new one is claimed. There is no mechanism to automatically spill over into a second account once the first reaches its cluster capacity.

As GAs grow, this becomes a scaling bottleneck. A solution is needed to automatically distribute clusters across multiple hyperscaler accounts once a configurable per-account limit is reached.

## Decision

### Rollout Strategy

This feature can be enabled gradually using an allowlist. With the allowlist, you can control which GAs get multi-account support. Alternatively, a simple boolean flag can enable/disable the feature for all GAs.

| Mode        | Config                                | Use Case                                               | Decision |
|-------------|---------------------------------------|--------------------------------------------------------|:--------:|
| Simple Flag | `enabled: true/false`                 | Enable/disable for all GAs (allowlist not implemented) | ❌       |
| Allowlist   | `allowlist: [GA-1, GA-2, GA-3]`       | Start with specific GAs                                | ✅       |
| All GAs     | `allowlist: ["*"]`                    | Enable for everyone                                    | ✅       |
| Disabled    | `allowlist: []` or no config provided | Feature off                                            | ✅       |

**Decision rationale:** Allowlist provides all required modes with a single field: specific GAs, global enablement (`["*"]`), and disabled (`[]`). Simpler to implement and allows testing on selected GAs.

### Cluster Limits per Hyperscaler Account

Each hyperscaler provider type should have a configurable maximum number of clusters. Once the limit is reached, new clusters are provisioned in a different account. A default limit is used when a provider limit is not specified in the configuration. Specific providers can override this default value if needed.

| Provider | Max clusters per account |
|----------|--------------------------|
| AWS | 200 |
| GCP | TBD |
| Azure | TBD |
| OpenStack | TBD |
| AliCloud | TBD |

### Hyperscaler Account Selection

When a global account has multiple hyperscaler accounts with available capacity, we need a strategy to choose which one to use for the next cluster.

| Strategy             | Behavior                          | Benefit                                                                    | Decision |
|----------------------|-----------------------------------|----------------------------------------------------------------------------|:--------:|
| Fill most-populated  | Fill account with most clusters   | Empties least-populated accounts so that they become empty for reclamation | ✅       |
| Fill-first           | Fill most recently assigned       |                 ?                                                          | ❌       |
| Round-robin          | Distribute evenly                 | Even distribution                                                          | ❌       |

**Decision rationale:** The fill-most-populated strategy offers the greatest chance that an account will have no clusters, and consequently, the cleanup job will reclaim it.


### Backward Compatibility

Hyperscaler accounts with clusters above the provider limit continue working. New clusters use a different account.

**Example:** GA has 250 clusters (limit: 180)
- Existing 250: Keep working
- New 251: New account
- After 71 deprovisioned: 179 total -> available again

### Cluster Counting per Account

To select the right hyperscaler account during provisioning, we must know how many clusters each account (CredentialsBinding) currently has. We must choose a method to determine this count.

| Method           | Pros | Cons | Decision |
|------------------|------|------|----------|
| Gardener API     | - Exact number of clusters on a hyperscaler account<br>- Existing implementation covers a lot of it<br>- Simple<br>- Gardener has knowledge about Shoots created w/o KEB and consuming subscriptions | - Performance (reading 2k Shoots takes approximately 5s and returns 35MB of payload)<br>- Puts additional load on APIServer | ❌ |
| Database         | - Querying is fast<br>- Minimal resource consumption | - May have inconsistent data (compared to Gardener)<br>- Some Shoots might be directly created in Gardener and KEB has no knowledge about it | ✅ |
| Runtime CRs      | | - May have inconsistent data (compared to Gardener)<br>- Shoots created manually directly in Gardener wouldn't be reflected in Runtime CRs | ❌ |

**Decision rationale:** Database approach is chosen for performance and resource efficiency. While the Gardener API provides the exact number of clusters on a hyperscaler account, its performance characteristics would impact provisioning speed and put unnecessary stress on the APIServer. The concern about Shoots created outside KEB is not applicable to production environments, as manual Shoot creation outside the dev environment should not take place. In production, all Shoots must be created through KEB, ensuring data consistency.

### Provisioning Flow

Existing HAP account selection rules remain unchanged. This feature adds the capability to use multiple CredentialsBindings for a given provider in one GA.

- When the feature is disabled (GA not in allowlist), the flow is the following:
  1. Existing HAP rules determine the labels used to select the CredentialsBinding.
  2. Select a CredentialsBinding using the defined labels.
  3. If none found, claim a new CredentialsBinding.
  4. Provision a cluster using the selected CredentialsBinding.

- When the feature is enabled (GA in allowlist),  the flow is the following:
  1. Existing HAP rules determine the labels used to select the CredentialsBinding.
  2. Find all CredentialsBindings matching the defined labels.
  3. If found, query the database for cluster counts, select the most-populated account that is still below the limit; if none found, claim a new CredentialsBinding.
  4. If all accounts are at the limit, claim a new CredentialsBinding.
  5. Provision a cluster using the selected CredentialsBinding.

  **Example:** Global account with AWS limit = 200
  - 150 clusters on CredentialsBinding-A -> provision on CredentialsBinding-A (below limit)
  - 200 clusters on CredentialsBinding-A -> claim CredentialsBinding-B, provision on CredentialsBinding-B
  - 200 on CredentialsBinding-A, 150 on CredentialsBinding-B -> provision on CredentialsBinding-B (fill-most-populated)
  - 199 on CredentialsBinding-A, 150 on CredentialsBinding-B -> provision on CredentialsBinding-A (still below limit, fill-most-populated)

### Configuration

```yaml
hap:
  multiHyperscalerAccount:
    # assigning multiple hyperscaler accounts per global account when capacity limits are reached
    # - Empty array [] = feature disabled
    # - Specific GAs = enabled only for listed global accounts
    # - ["*"] = enabled for all global accounts
    allowedGlobalAccounts: ["*"]

    # Maximum clusters per hyperscaler account (applies only when allowedGlobalAccounts is not empty)
    # When a CredentialsBinding reaches this limit, new clusters use a different account
    limits:
      # Default limit used when a provider limit is not specified
      # This allows new providers to be supported without configuration updates
      default: 3
      # Provider-specific overrides (optional)
      # Only define these if a provider requires a different limit than the default
      aws: 180
      gcp: 135
      openstack: 100
      alicloud: 100
    
    # Account selection strategy (field will be added if multiple strategies are implemented)
    # strategy: "fill-most-populated"  # Possible options: "fill-most-populated", "fill-first", "round-robin"
```

## Consequences

GAs can scale beyond the capacity of a single hyperscaler account without manual intervention. The gradual rollout via allowlist minimizes risk during adoption. Existing HAP selection rules remain unchanged; the feature layers on top transparently.

Database-based counting is performant and avoids additional load on the Gardener APIServer. The fill-most-populated strategy increases the likelihood that accounts can be fully reclaimed by the cleanup job over time.

The database may be slightly inconsistent with actual Gardener state if Shoots are created outside KEB, though this should not happen in production environments.

### Metrics

The following metrics are introduced to observe the feature:

- `keb_credentials_bindings_per_ga` - CredentialsBindings per GA (1 CredentialsBinding = 1 hyperscaler account)
- `keb_shoots_per_credentials_binding` - Shoots per CredentialsBinding
- `keb_available_credentials_bindings` - Unclaimed CredentialsBindings in a pool

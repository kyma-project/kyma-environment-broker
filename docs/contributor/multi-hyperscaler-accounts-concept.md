# Multi-Hyperscaler Accounts per Global Account

## Problem

One Global Account can only use one hyperscaler account per provider for regular provisioning, limiting cluster count to the account's capacity.

> ### Note: 
> Multiple accounts per Global Account (GA) are already supported for special cases (e.g., normal AWS and EU Access AWS cluster in the same GA). However, automatic assignment based on capacity limits is not yet implemented.

## Solution

Automatically assign multiple accounts per GA when capacity limit is reached.

**Current provisioning:**
- Find CredentialsBinding with `tenantName={GA}`
- Use it to provision the cluster
- If none exists, claim new one

**New provisioning:**
- Find ALL CredentialsBindings with `tenantName={GA}`
- Count clusters in each
- Use CredentialsBinding below limit (each CredentialsBinding = one hyperscaler account)
- If all full, claim new one

## Design

### 1. Rollout Strategy

This feature can be enabled gradually using an allowlist, letting you control which GAs get multi-account support. Alternatively, a simple boolean flag could enable/disable it for all GAs.

| Mode       | Config                        | Use Case                      | Decision |
|------------|------------------------------|-------------------------------|----------|
| Simple Flag | `enabled: true/false`       | Enable/disable for all GAs (allowlist not implemented) | ❌ |
| Allowlist  | `allowlist: [GA-1, GA-2, GA-3]` | Start with specific GAs        | 	✅ |
| All GAs    | `allowlist: ["*"]`           | Enable for everyone            | 	✅ |
| Disabled   | `allowlist: []` or no config provided | Feature off                 | 	✅ |

**Decision rationale:** Allowlist provides all required modes with a single field: specific GAs, global enablement (`["*"]`), and disabled (`[]`). Simpler to implement and allows testing on selected GAs.

### 2. Cluster Limits

Each hyperscaler provider type should have a configurable maximum number of clusters. Once the limit is reached, new clusters are provisioned in a different account. A default limit is used when a provider limit is not specified in the configuration. Specific providers can override this default value if needed.

| Provider | Max clusters per account|
|----------|--------------|
| AWS | 200 |
| GCP | TBD |
| Azure | TBD |
| OpenStack | TBD |
| AliCloud | TBD |

### 3. Hyperscaler Account Selection

When a Global Account has multiple hyperscaler accounts with available capacity, we need a strategy to choose which one to use for the next cluster.

| Strategy             | Behavior                          | Benefit                        | Decision     |
|----------------------|-----------------------------------|-------------------------------|---------------|
| Fill most-populated  | Fill account with most clusters   | Empties least-populated faster | 	✅ |
| Fill-first           | Fill most recently assigned       |                 ?              | ❌ |
| Round-robin          | Distribute evenly                 | Even distribution              | ❌ |

**Decision rationale:** Fill most-populated strategy has the greatest chance that an account will have no clusters, allowing the cleanup job to reclaim it.


### 4. Backward Compatibility

Accounts with clusters above the provider limit continue working. New clusters use different account.

**Example:** GA has 250 clusters (limit: 180)
- Existing 250: Keep working
- New 251: New account
- After 71 deprovisioned: 179 total -> available again

### 5. Cluster Counting per Account

To select the right hyperscaler account during provisioning, we need to know how many clusters each account (CredentialsBinding) currently has. We must choose a method to determine this count.

| Method           | Pros | Cons | Decision |
|------------------|------|------|----------|
| Gardener API     | - Existing implementation covers a lot of it<br>- Simple<br>- Gardener has knowledge about shoots created w/o KEB and consuming subscriptions | - Performance (reading 2k shoots takes approximately 5s and returns 35MB of payload)<br>- Puts additional load on apiserver<br>- May have inconsistent data | ❌ |
| Database         | - Querying is fast<br>- Minimal resource consumption | - May have inconsistent data<br>- Some shoots might be directly created in Gardener and KEB has no knowledge about it | ✅ |
| Runtime CRs      | | - May have inconsistent data<br>- Some shoots might be directly created in Gardener and KEB has no knowledge about it | ❌ |

**Decision rationale:** Database approach is chosen for performance and resource efficiency. While the Gardener API provides complete data, its performance characteristics would impact provisioning speed and put unnecessary stress on the apiserver. The concern about shoots created outside KEB is not applicable to production environments, as manual shoot creation outside the dev environment should not take place. In production, all shoots must be created through KEB, ensuring data consistency.

## Integration with Current HAP Implementation

Existing HAP account selection rules remain unchanged. This feature adds the capability to use multiple CredentialsBindings for a given provider in one GA.

**Provisioning flow:**

**When feature is disabled (GA not in allowlist):**
1. Existing HAP rules determine the labels used to select the CredentialsBinding
2. Find CredentialsBinding using the defined labels
3. If none found: claim new CredentialsBinding
4. Provision cluster using selected CredentialsBinding

**When feature is enabled (GA in allowlist):**
1. Existing HAP rules determine the labels used to select the CredentialsBinding
2. Find CredentialsBindings using the defined labels
3. If none found: claim new CredentialsBinding
4. If found: count clusters in each using database query, select most populated below limit
5. If all at limit: claim new CredentialsBinding
6. Provision cluster using selected CredentialsBinding

**Example:** GlobalAccount with AWS limit = 200
- 150 clusters on CredentialsBinding-A -> provision on CredentialsBinding-A (below limit)
- 200 clusters on CredentialsBinding-A -> claim CredentialsBinding-B, provision on CredentialsBinding-B
- 200 on CredentialsBinding-A, 150 on CredentialsBinding-B -> provision on CredentialsBinding-B (fill-most-populated)
- 199 on CredentialsBinding-A, 150 on CredentialsBinding-B -> provision on CredentialsBinding-A (still below limit, fill-most-populated)

## Configuration

```yaml
hap:
  multiHyperscalerAccount:
    # assigning multiple hyperscaler accounts per Global Account when capacity limits are reached
    # - Empty array [] = feature disabled
    # - Specific GAs = enabled only for listed Global Accounts
    # - ["*"] = enabled for all Global Accounts
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

## Metrics

- `keb_credentials_bindings_per_ga` - CredentialsBindings per GA (1 CredentialsBinding = 1 hyperscaler account)
- `keb_shoots_per_credentials_binding` - Shoots per CredentialsBinding
- `keb_available_credentials_bindings` - Unclaimed CredentialsBindings in pool

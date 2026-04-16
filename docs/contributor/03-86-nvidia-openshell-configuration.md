<!--{"metadata":{"publish":false}}-->

# NVIDIA Open Shell Configuration

NVIDIA Open Shell is a per-global-account feature that enables NVIDIA Open Shell support on Kyma runtime clusters. When enabled for a global account, KEB sets the `EnableNvidiaOpenshell` flag on the `Runtime` custom resource during provisioning. Infrastructure Manager then configures the cluster accordingly.

Runtimes belonging to global accounts that are **not** in the allowlist are not affected — the flag is set to `false` and no NVIDIA-specific configuration is applied.

## Configuration

The feature is controlled by an allowlist stored in a YAML file mounted into the KEB pod.

### Helm Values

Configure the allowlist in `values.yaml`:

```yaml
# List of global account IDs for which NVIDIA Open Shell is enabled.
openShellWhitelistedGlobalAccountIds: |-
  whitelist:
    - <global-account-id-1>
    - <global-account-id-2>
```

The default value is an empty list (`whitelist:` with no entries), meaning the feature is disabled for all global accounts.


### Allowlist File Format

The YAML file must follow this structure:

```yaml
whitelist:
  - <global-account-id-1>
  - <global-account-id-2>
```

## Behavior

During provisioning, KEB checks whether the global account ID from the request context is present in the allowlist:

- **Account is in the allowlist** — `EnableNvidiaOpenshell` is set to `true` on the `Runtime` CR. Infrastructure Manager provisions the cluster with NVIDIA Open Shell support.
- **Account is not in the allowlist** — `EnableNvidiaOpenshell` is set to `false`. The runtime is provisioned without any NVIDIA-specific configuration.

The check happens in the `CreateRuntimeResource` provisioning step and applies to all new provisioning operations.

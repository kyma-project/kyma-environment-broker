<!--{"metadata":{"publish":false}}-->

# Operation Blocklist

## Overview

You can configure Kyma Environment Broker (KEB) to block specific operations (provisioning, deprovisioning, update, plan upgrade) for selected service plans. When a blocked operation is attempted, KEB rejects the request with an HTTP 400 error and the configured message.

## Configuration

The **APP_OPERATION_BLOCKLIST_FILE_PATH** environment variable points to the blocklist defined in a YAML file. In the Helm chart, set the **operationBlocklist** value.

```yaml
operationBlocklist: |-
  provision:
    - '"Provisioning of the trial plan is currently blocked","plan=trial"'
  deprovision:
    - '"Deprovisioning of the trial plan is currently blocked","plan=trial"'
```

The file is served from the existing `/config` volume through the `kcp-kyma-environment-broker` ConfigMap.

If you don't set **operationBlocklist** or leave it empty, no operations are blocked.

## Rule Format

Each rule is a compact string with up to two quoted tokens separated by a comma:

```
'"<message>"'
'"<message>","plan=<plan1>,<plan2>"'
```

Use the following rule components:

- message — required, non-empty text string returned to the caller when the rule matches. Supports the `{plan}` placeholder, which is replaced with the actual plan name at runtime.
- plan filter — required comma-separated list of plan names to match. A rule without a plan filter is a no-op.

### Examples

```yaml
# Block provisioning for all plans
provision: '"Provisioning is temporarily disabled"'
```

> ### Note:
> This rule has no plan filter and is therefore a no-op. A plan filter is required for a rule to take effect.

```yaml
# Block provisioning for trial only
provision: '"Provisioning of the {plan} plan is blocked","plan=trial"'

# Block update for multiple plans
update:
  - '"Updates are blocked for {plan}","plan=trial,free"'

# Block plan upgrade and deprovision for trial
planUpgrade: '"Plan upgrade is not allowed for {plan}","plan=trial"'
deprovision: '"Deprovisioning is blocked for {plan}","plan=trial"'
```

## Supported Operations

| Key | Operation blocked |
|---|---|
| **provision** | New instance provisioning |
| **update** | Instance update |
| **planUpgrade** | Plan upgrade |
| **deprovision** | Instance deprovisioning |

## Validation

KEB validates the blocklist at startup. The following configurations are rejected with an error:

| Invalid input | Error |
|---|---|
| `'""'` | Empty message |
| `'"msg","plan="'` | Empty plan filter |
| `'"msg","plan=aws,,gcp"'` | Empty segment in plan list |
| `'"msg",'` | Trailing comma |
| Unknown top-level key (for example, `planUpgarde`) | Typo detection |
| Unknown plan name (for example, `trail`) | Caught by plan validator at startup |

A rule with no plan filter (`'"msg"'`) or an empty string rule (`''`) or an empty key (for example, `provision:`) is a no-op and does not cause an error.

## Plan Names

Valid plan names are the same as those enabled using **broker.enablePlans**, for example, `aws`, `azure`, `gcp`, `trial`, `free`. A typo in a plan name (for example, `trail` instead of `trial`) causes a startup error.

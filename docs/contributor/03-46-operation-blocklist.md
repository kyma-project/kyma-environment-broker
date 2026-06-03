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

Each rule is a compact string with quoted tokens separated by commas:

```
'"<message>"'
'"<message>","plan=<plan1>,<plan2>"'
'"<message>","plan=<plan1>,<plan2>","GA=<id1>,<id2>"'
'"<message>","plan=<plan1>,<plan2>","GA!=<id1>,<id2>"'
```

### Tokens

The following tokens define the rule parameters:

* **{message}** — required. Non-empty text returned to the caller when the operation is blocked. Supports the `{plan}` and `{globalAccount}` placeholders, which KEB replaces with the actual plan name and GlobalAccount ID at runtime.

* `plan=<plan1>,<plan2>` — required when any GA filter is present. Comma-separated list of plan names. The operation is blocked only if its plan is one of the listed plans.
  * A single plan: `plan=trial`
  * Multiple plans: `plan=trial,aws` — blocks both `trial` and `aws`

* `GA=<id1>,<id2>` — optional. Only operations from the listed GlobalAccounts are blocked; all other GlobalAccounts are allowed.
  * A single account: `GA=<id>` — operations from this GlobalAccount are blocked; all others are allowed.
  * Multiple accounts: `GA=<id1>,<id2>` — operations from `id1` or `id2` are blocked; all others are allowed.

* `GA!=<id1>,<id2>` — optional. All GlobalAccounts except the listed ones are blocked.
  * A single exemption: `GA!=<id>` — all GlobalAccounts except `id` are blocked.
  * Multiple exemptions: `GA!=<id1>,<id2>` — all GlobalAccounts except `id1` and `id2` are blocked.

> ### Note:
> GlobalAccount ID matching is case-insensitive — `GA=7F3A9B1C-12D4-4E5F-A678-9B0CDE123456` matches `7f3a9b1c-12d4-4e5f-a678-9b0cde123456`.

> ### Note:
> `GA=` and `GA!=` require `plan=` to be present. A rule that blocks based on GlobalAccount alone, regardless of plan, is not supported — a GA filter without `plan=` is rejected at startup.

> ### Note:
> A rule with only a message and no filters is a no-op and does not cause an error.

### Examples

```yaml
# No plan filter — this rule is a no-op and does not block anything
provision: '"Provisioning is temporarily disabled"'
```

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

### Filter Semantics

All filters in a single rule are combined with **AND** — the operation is blocked only when every filter condition is satisfied:

| `plan=` | `GA=` / `GA!=` | blocks when |
|---|---|---|
| `plan=trial` | — | plan is `trial` |
| `plan=trial` | `GA=X` | plan is `trial` **and** GA is X |
| `plan=trial` | `GA=X,Y` | plan is `trial` **and** GA is X or Y |
| `plan=trial` | `GA!=X` | plan is `trial` **and** GA is not X |
| `plan=trial` | `GA!=X,Y` | plan is `trial` **and** GA is neither X nor Y |

`GA=` — block only the listed GAs; all others are allowed.

`GA!=` — block everyone except the listed GAs (broad block with exemptions).

## Multiple Rules

Rules within an operation type are evaluated in order. **The first matching rule wins** — evaluation stops and its message is returned. Rules that do not match are skipped.

This means more specific rules (with `GA=`) should come **before** broader ones (with `plan=` only).

### Patterns

**Block specific accounts, allow everyone else:**

```yaml
provision:
  - '"Blocked for GA1 and GA2","plan=trial","GA=ga-1,ga-2"'
```

| plan | GA | result |
|---|---|---|
| `trial` | `ga-1` | blocked — "Blocked for GA1 and GA2" |
| `trial` | `ga-2` | blocked — "Blocked for GA1 and GA2" |
| `trial` | anything else | allowed |

**Block everyone except specific accounts (broad block with exemptions):**

```yaml
provision:
  - '"Trial plan is temporarily blocked","plan=trial","GA!=ga-exempt-1,ga-exempt-2"'
```

| plan | GA | result |
|---|---|---|
| `trial` | `ga-exempt-1` | allowed |
| `trial` | `ga-exempt-2` | allowed |
| `trial` | anything else | blocked — "Trial plan is temporarily blocked" |

**Catch-all after specific rules:**

```yaml
provision:
  - '"VIP account","plan=trial","GA=ga-vip"'   # blocks only ga-vip
  - '"Trial plan is temporarily blocked for {plan}","plan=trial"' # catch-all for everyone else
```

| plan | GA | result |
|---|---|---|
| `trial` | `ga-vip` | blocked — "VIP account" (rule 1 matches) |
| `trial` | anything else | blocked — "Trial plan is temporarily blocked for trial" (rule 1 doesn't match, rule 2 does) |
| `aws` | anything | allowed (neither rule matches) |

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
| `'"msg","GA="'` | Empty GA value |
| `'"msg","GA!="'` | Empty GA value |
| `'"msg","GA=ga-1,,ga-2"'` | Empty segment in GA list |
| `'"msg","GA!=ga-1,,ga-2"'` | Empty segment in GA list |
| `'"msg","GA=X"'` | GA filter without `plan=` |
| `'"msg","GA!=X"'` | GA filter without `plan=` |
| `'"msg",'` | Trailing comma |
| Unknown top-level key (for example, `planUpgarde`) | Typo detection |
| Unknown plan name (for example, `trail`) | Caught by plan validator at startup |

A rule with only a message (`'"msg"'`), an empty string rule (`''`), or an empty key (for example, `provision:`) is a no-op and does not cause an error.

> ### Note:
> `GA=` and `GA!=` values are **not** validated at startup (unlike plan names). If the GlobalAccount ID in `GA=` does not match any real account, the rule never triggers — no one is blocked by it. If the ID in `GA!=` does not match any real account, the rule blocks everyone for that plan (no one is exempted).

## Plan Names

Valid plan names are the same as those enabled using **broker.enablePlans**, for example, `aws`, `azure`, `gcp`, `trial`, `free`. A typo in a plan name (for example, `trail` instead of `trial`) causes a startup error.

## Known Limitations

### Subaccount Move and the Update Check

When a subaccount is moved to a different GlobalAccount, the incoming `ersContext.GlobalAccountID` differs from the value stored on the instance. The `update` blocklist check runs **before** `handleSubaccountMoveRequest` applies the new GlobalAccount ID to the instance. As a result, the check is evaluated against the pre-move `instance.GlobalAccountID`:

* A `GA=<new-ga>` rule does **not** block the update request that performs the move — the new GA is not yet set on the instance at the time of the check.
* A `GA!=<old-ga>` rule that exempts the old GA correctly allows the request.

After the move completes, the instance is stored with the new `GlobalAccountID` and all subsequent operations are evaluated against the new value.

For all other update requests (not subaccount moves), `ersContext.GlobalAccountID` is empty and the check correctly uses `instance.GlobalAccountID`.

## Extending the Rule Format

The rule format is designed for extensibility. Future filters follow the same token pattern:

* Positive filter (`key=value`): rule applies only when the attribute matches
* Negation filter (`key!=value`): rule does not apply when the attribute matches

To add a SubAccount filter (`SA=` / `SA!=<subAccountID>`), extend `OperationContext` and `Rule` in `internal/blocklist/blocklist.go` following the existing `GA=` / `GA!=` pattern.

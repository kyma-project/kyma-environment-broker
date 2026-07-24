# Stress Testing KEB

This document describes how to run a bulk provisioning stress test against KEB installation using the `keb.py` utility script.

## Prerequisites

* `python3` with the `requests` package:

  ```bash
  pip install requests
  ```

* KEB port-forwarded to `localhost:8080`:

  ```bash
  kubectl port-forward -n kcp-system deployment/kcp-kyma-environment-broker 8080:8080
  ```

## Configuration

Before running the stress test, apply the following configuration overrides.

### Allow Multiple Trial Instances per Global Account

By default, KEB restricts each global account to one active trial instance. To allow provisioning multiple trial instances for the same global account, set the following value:

```yaml
broker:
  onlySingleTrialPerGA: "false"
```

### Whitelist the Subaccount to Skip Quota Checks

To allow the test subaccount to provision beyond its quota limits, add it to the quota whitelist:

```yaml
quotaWhitelistedSubaccountIds: |-
  whitelist:
    - <subaccount-id>
```

The default subaccount ID used by `keb.py` is `github-actions-keb-integration`.

## Scenario

The stress test consists of three steps:

1. **Provision** — create N trial instances in bulk.
2. **Monitor** — poll instance states until all are `succeeded` or `failed`.
3. **Deprovision** — clean up all provisioned instances.

## Step 1: Provision Instances

Run the following command to provision N trial instances for a given global account:

```bash
python3 keb.py provision <N> --global-account-id <global-account-id>
```

The instance IDs are saved to a timestamped file, for example `instances_20260724_143900.txt`.

## Step 2: Monitor Instances

Poll instance states until all are `succeeded` or `failed`:

```bash
python3 keb.py monitor instances_<timestamp>.txt --interval 30
```

## Step 3: Deprovision Instances

Deprovision all instances from the instances file:

```bash
python3 keb.py deprovision instances_<timestamp>.txt
```

## Full Example

```bash
# Provision 100 trial instances
python3 keb.py provision 100 --global-account-id my-global-account-id

# Monitor until all instances succeed or fail
python3 keb.py monitor $(ls -t instances_*.txt | head -1) --interval 30

# Deprovision all instances
python3 keb.py deprovision $(ls -t instances_*.txt | head -1)
```

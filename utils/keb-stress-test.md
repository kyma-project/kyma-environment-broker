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

### Allowlist the Subaccount to Skip Quota Checks

To allow the test subaccount to provision beyond its quota limits, add it to the quota allowlist:

```yaml
quotaWhitelistedSubaccountIds: |-
  whitelist:
    - <subaccount-id>
```

The default subaccount ID used by `keb.py` is `github-actions-keb-integration`.

## Procedure

 1. To provision N trial instances for a given global account, run the following command:

    ```bash
    python3 keb.py provision <N> --global-account-id <global-account-id> [--concurrent <threads>]
    ```

    The `--concurrent` flag controls how many provisioning requests are in-flight at the same time (default: `1` — sequential). For large N, increase this to reduce total submission time. 

    The instance IDs are saved to a timestamped file, for example `instances_20260724_143900.txt`.

 2. Monitor instances: Poll instance states until all are `succeeded` or `failed`.

    ```bash
    python3 keb.py monitor instances_<timestamp>.txt --interval 30
    ```

 3. Deprovision all instances from the instances file.

    ```bash
    python3 keb.py deprovision instances_<timestamp>.txt
    ```

### Full Example

```bash
# Provision 100 trial instances (5 concurrent requests)
python3 keb.py provision 100 --global-account-id my-global-account-id --concurrent 5

# Monitor until all instances succeed or fail
python3 keb.py monitor $(ls -t instances_*.txt | head -1) --interval 30

# Deprovision all instances
python3 keb.py deprovision $(ls -t instances_*.txt | head -1)
```

<!--{"metadata":{"requirement":"RECOMMENDED","type":"INTERNAL","category":"CONFIGURATION"}}-->

# KEB: Internal-Only Machine Types

> ### Note: 
> This change is recommended. Without configuring **internalOnlyMachines**, machine types intended for internal SAP use — such as GPU-equipped instances — remain accessible to external customers.

## What's Changed

KEB now supports the **internalOnlyMachines** field in the plan configuration. The machine types listed there are blocked for requests from external customers (carrying an external license type) but remain available to internal SAP users.

The restriction applies to the following:
- The main machine type in a provisioning or update request.
- Machine types used in additional worker node pools.

If an external customer specifies a restricted machine type, the request is rejected with an error message that lists the restricted machine types and the worker pool names where they appear.

At startup, KEB logs a warning for any **internalOnlyMachines** entry that is redundant (already covered by a shorter prefix) or unmatched (does not correspond to any machine in **regularMachines** or **additionalMachines**). These warnings are informational and do not prevent KEB from starting.

## Procedure

Add **internalOnlyMachines** to the relevant plans in the plan configuration. Each entry is either a family prefix (matches all machine types starting with that string) or a fully-qualified machine type name (matches only that exact type).

```yaml
plansConfiguration:
  aws:
    regularMachines:
      - "m6i.large"
      - "m6i.xlarge"
    additionalMachines:
      - "g6.xlarge"
      - "g6.2xlarge"
      - "g4dn.xlarge"
    internalOnlyMachines:
      - "g6"           # family prefix — restricts all g6.* sizes
      - "g4dn.xlarge"  # fully-qualified — restricts only this specific size
```

For more configuration details, see [Internal-Only Machine Types](https://github.com/kyma-project/kyma-environment-broker/blob/main/docs/contributor/03-73-internal-only-machines.md).

## Post-Update Steps

After updating the configuration, perform the following steps:

1. Check KEB startup logs for configuration warnings. KEB logs a warning for each misconfigured **internalOnlyMachines** entry. Look for lines similar to the following:

   - Redundant entry (already covered by a shorter prefix in the same list):
     ```
     internalOnlyMachines entry "g6.xlarge" is redundant in plan "aws" — already covered by prefix "g6"
     ```

   - Unmatched entry (does not correspond to any machine in **regularMachines** or **additionalMachines**):
     ```
     internalOnlyMachines entry "g6.unknown" in plan "aws" does not match any machine type in regularMachines or additionalMachines
     ```

   These warnings are informational, and they don't prevent KEB from starting. They only indicate entries that either have no effect or overlap with a prefix and should be corrected.

2. Verify request enforcement: Confirm that external provisioning and update requests using the restricted machine types are rejected, and that internal requests continue to succeed.

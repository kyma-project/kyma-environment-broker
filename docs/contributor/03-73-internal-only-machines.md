<!--{"metadata":{"publish":true}}-->

# Internal-Only Machine Types

Some machine types — such as GPU-equipped instances — are not available to all customers. The **internalOnlyMachines** configuration restricts these machine types to internal SAP users and prevents external customers from using them.

## Configuration

Define **internalOnlyMachines** as a list of machine type entries in the plan configuration:

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

Each entry is either a family prefix or a fully-qualified machine type name. A machine type is considered internal-only if it matches an entry exactly or starts with one of the listed prefixes. In the example above, `g6` matches `g6.xlarge`, `g6.2xlarge`, and any other machine type starting with `g6`. A fully-qualified entry, such as `g4dn.xlarge,` restricts only that specific size.

## Access Control

The check is applied only when the provisioning or update request carries an external customer license type. Internal SAP users are not subject to this restriction.

If an external customer includes a restricted machine type — either as the main machine type or inside an additional worker node pool — the request is rejected with an error message that lists all restricted machine types and the worker pool names where they are used.

## Startup Validation

At startup, KEB validates the **internalOnlyMachines** configuration and logs a warning for each of the following issues:

- Redundant entry: A fully-qualified name is already covered by a shorter prefix in the same list. For example, `g6.xlarge` is redundant when `g6` is also listed.
- Unmatched entry: An entry does not match any machine type in **regularMachines** or **additionalMachines**.

These warnings are informational only and do not prevent KEB from starting.

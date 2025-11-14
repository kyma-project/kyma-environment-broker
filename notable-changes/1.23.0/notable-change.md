<!--{"metadata":{"requirement":"RECOMMENDED","type":"INTERNAL","category":"CONFIGURATION","additionalFiles":0}}-->

# Updating Kyma Environment Broker: Dual-Stack Networking Support

> [!NOTE]
> This is a recommended change. To enable the new dual-stack networking feature, update the Kyma Environment Broker (KEB) provider configuration.

## Prerequisites

- KEB is configured to use a supported cloud provider (AWS or GCP).

## What's Changed

A new dual-stack networking feature has been added to KEB, allowing Kyma runtimes to support both IPv4 and IPv6 protocols simultaneously. This feature is currently supported for AWS and GCP providers and is configured at the provider level to become available in the service catalog when enabled.

## Procedure

1. Open the KEB configuration file.
2. Locate the provider configuration under `providersConfiguration`.
3. Add the dual-stack configuration for supported providers. See the following example:

    - Updated provider configuration with dual-stack support:
    
        ```yaml
        providersConfiguration:
          aws:
            dualStack: true
            machines:
              # ... existing machine configurations
            regions:
              # ... existing region configurations
          gcp:
            dualStack: true
            machines:
              # ... existing machine configurations
            regions:
              # ... existing region configurations
        ```

4. Save and apply the updated configuration.
5. Refresh broker details in one BTP region using the XRS APIs in the ERS registry.

## Impact on Provisioning

With this new feature, dual-stack networking capabilities are determined by the cloud provider configuration:

- **Dual-stack enabled providers**: When `dualStack: true` is set, the `dualStack` parameter becomes available in the provisioning request's networking section
- **Dual-stack disabled providers**: The `dualStack` parameter is not available in the service schema

Example provisioning request using the new dual-stack networking feature:

```json
{
  "parameters": {
    "name": "my-cluster",
    "region": "eu-central-1",
    "networking": {
      "nodes": "10.250.0.0/20",
      "dualStack": true
    }
  }
}
```

## Post-Update Steps

1. Verify that the dual-stack option appears in the service catalog for plans using providers with `dualStack: true`
2. Verify that the dual-stack option appears in SAP BTP cockpit
3. Test provisioning with dual-stack networking enabled to ensure the new feature works correctly
4. Check that dual-stack configuration is properly applied to the runtime resources

For more information about configuring dual-stack networking in Kyma Environment Broker, see [Dual-Stack Configuration](../../docs/contributor/03-85-dual-stack-configuration.md).
For information about using dual-stack networking when provisioning Kyma instances, see [Custom Networking Configuration](../../docs/user/04-30-custom-networking-configuration.md).
<!--{"metadata":{"requirement":"MANDATORY","type":"EXTERNAL","category":"FEATURE","additionalFiles":0}}-->

# Updating Kyma Environment Broker: Dual-Stack Networking Support

> [!WARNING]
> This is a mandatory change. You must update the Kyma Environment Broker (KEB) provider configuration to enable the new dual-stack networking feature.

## Prerequisites

- KEB is configured to use a supported cloud provider (Amazon Web Services or Google Cloud).

## What's Changed

A new dual-stack networking feature has been added to KEB, allowing Kyma runtimes to support both IPv4 and IPv6 protocols simultaneously. This feature is supported for Amazon Web Services and Google Cloud providers. It is configured at the provider level to become available in the SAP BTP cockpit.

## Procedure

1. Open the KEB configuration file.
2. Locate the provider configuration under `providersConfiguration`.
3. Add the dual-stack configuration for supported providers. See the following example of an updated provider configuration with dual-stack support:
    
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
5. Refresh broker details using the xRS APIs in Environment Registry Service (ERS).

## Impact on Provisioning

With this new feature, dual-stack networking capabilities are determined by the cloud provider configuration in the following way:

- For providers with dual stack enabled: When `dualStack: true` is set, the **dualStack** parameter becomes available in the provisioning request's networking section.
- For providers with dual stack disabled: The **dualStack** parameter is not available in the networking section.

See an example provisioning request using the new dual-stack networking feature.

```json
{
  "parameters": {
    "name": "my-cluster",
    "region": "eu-central-1",
    "networking": {
      "dualStack": true,
      "nodes": "10.250.0.0/20"
    }
  }
}
```

## Post-Update Steps

Verify that the dual-stack option appears in the SAP BTP cockpit for supported plans.

For more information about configuring dual-stack networking in Kyma Environment Broker, see [Dual-Stack Configuration](https://github.com/kyma-project/kyma-environment-broker/blob/main/docs/contributor/03-85-dual-stack-configuration.md).
For information about using dual-stack networking when provisioning Kyma instances, see [Custom Networking Configuration](https://github.com/kyma-project/kyma-environment-broker/blob/main/docs/user/04-30-custom-networking-configuration.md).

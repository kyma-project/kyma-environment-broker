<!--{"metadata":{"requirement":"RECOMMENDED","type":"INTERNAL","category":"CONFIGURATION"}}-->

# Updating Kyma Environment Broker: Zones Discovery

> [!NOTE]
> This is a recommended change. To enable Zones Discovery for AWS, update the Kyma Environment Broker (KEB) configuration.

## Prerequisites

- KEB is configured to use the AWS provider.

## What's Changed

With the [Zones Discovery](https://github.com/kyma-project/kyma-environment-broker/blob/main/docs/contributor/03-55-zones-discovery.md) feature, KEB can dynamically retrieve available zones from the hyperscaler (currently only AWS) instead of relying on statically configured zones in `providersConfiguration`.

## Procedure

1. Open the KEB configuration file.
2. Locate the AWS provider configuration under `providersConfiguration.aws`.
3. Replace the old static configuration with the new simplified format. See the following examples:

    - Static zones configuration
    
        ```yaml
        providersConfiguration:
          aws:
            regionsSupportingMachine:
              g6:
                us-west-2:
                eu-central-1: [a, b]
                ap-south-1: [b]
                us-east-1: [a, b, c, d]
              g4dn:
                eu-central-1:
                eu-west-2:
                us-east-1:
                ap-south-1:
                us-west-2: [a, b, c]
            regions:
              eu-central-1:
                displayName: eu-central-1 (Europe, Frankfurt)
                zones: [a, b, c]
              us-east-1:
                displayName: us-east-1 (US East, N. Virginia)
                zones: [a, b, c, d, f]
              eu-west-1:
                displayName: eu-west-1 (Europe, Ireland)
                zones: [a]
        ```

    - Zones Discovery configuration
    
        ```yaml
        providersConfiguration:
          aws:
            regionsSupportingMachine:
              g6:
                us-west-2:
                eu-central-1:
                ap-south-1:
                us-east-1:
              g4dn:
                eu-central-1:
                eu-west-2:
                us-east-1:
                ap-south-1:
                us-west-2:
            regions:
              eu-central-1:
                displayName: eu-central-1 (Europe, Frankfurt)
              us-east-1:
                displayName: us-east-1 (US East, N. Virginia)
              eu-west-1:
                displayName: eu-west-1 (Europe, Ireland)
            zonesDiscovery: true
        ```

4. Save and apply the updated configuration.

## Post-Update Steps

1. Monitor the KEB logs for any warnings about static zone configuration being ignored. See example log entries:

    ```json lines
    {"level":"WARN", "msg":"Provider aws has zones discovery enabled, but region us-west-2 is configured with 4 static zone(s), which will be ignored."}
    {"level":"WARN", "msg":"Provider aws has zones discovery enabled, but machine type g6 in region ap-south-1 is configured with 1 static zone(s), which will be ignored."}
    ```

2. Verify successful provisioning by checking that new runtimes are assigned zones dynamically. See example log entries:

    ```json lines
    {"level":"INFO", "msg":"Available zones for machine type m6i.large: [eu-central-1c eu-central-1b eu-central-1a]"}
    {"level":"INFO", "msg":"Zones for Kyma worker node pool: [eu-central-1c eu-central-1b eu-central-1a]"}
    ```

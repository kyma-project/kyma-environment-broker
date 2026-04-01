<!--{"metadata":{"requirement":"RECOMMENDED","type":"INTERNAL","category":"CONFIGURATION"}}-->

# KEB: Allowlist for Global Accounts with the Maximum Number of Pods

> ### Note:
> This is an optional change. To add the allowlist for global accounts with the maximum number of Pods, update the Kyma Environment Broker (KEB) configuration.

## What's Changed

Added support for configuring an allowlist of global account IDs that are allowed to use an increased maximum number of Pods. 
With the new **maxPodsWhitelistedGlobalAccountIds** configuration field, you can allowlist global accounts, enabling them 
to run up to 250 Pods across all worker node pools instead of the default limit.

Example configuration:

```yaml
maxPodsWhitelistedGlobalAccountIds: |-
  whitelist:
    - <global-account-id-1>
    - <global-account-id-2>
```

## Procedure

1. Open the KEB configuration file.
2. Add the **maxPodsWhitelistedGlobalAccountIds** section.
3. Add the desired global account IDs under **whitelist**.

    ```yaml
    maxPodsWhitelistedGlobalAccountIds: |-
      whitelist:
        - <global-account-id-1>
        - <global-account-id-2>
    ```

4. Save and apply the updated configuration.

## Post-Update Steps

1. Verify that the configuration has been applied successfully by checking the KEB logs.
2. Look for a log entry confirming the number of allowlisted global account IDs, for example:

    ```json lines
    {"level":"INFO", "msg":"Number of globalAccountIds for max pods: 2"}
    ```

3. Ensure that the reported number matches the number of global account IDs you added to the allowlist.

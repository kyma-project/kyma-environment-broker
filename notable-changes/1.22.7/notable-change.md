<!--{"metadata":{"requirement":"RECOMMENDED","type":"INTERNAL","category":"CONFIGURATION"}}-->

# Updating Kyma Environment Broker: Removal of Archiving and Cleaning Flags

> [!NOTE] 
> This is a recommended change. The `archiving` and `cleaning` feature flags are now removed because both processes are always active. 
> Without updating the configuration, the obsolete settings have no effect, but may cause confusion.

## What's Changed

The following configuration flags have been permanently removed from KEB:

```yaml
archiving:
  enabled:
  dryRun:
cleaning:
  enabled:
  dryRun:
```

The archiving and cleaning processes are now always active. The `enabled` and `dryRun` settings are ignored and should be removed.

## Procedure

1. Open the KEB configuration file.
2. If the following sections are present in your configuration, remove them.

    ```yaml
    archiving:
      enabled:
      dryRun:
    cleaning:
      enabled:
      dryRun:
    ```
   
3. Save and apply the updated configuration.

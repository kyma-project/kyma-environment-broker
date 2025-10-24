# Notable Changes

Notable changes refer to Kyma Environment Broker (KEB) updates requiring operator action. These changes can be classified along the following dimensions:

- Requirement:

  - Mandatory — Operator action is required for proper functionality.
  - Recommended — Operator action is recommended but not strictly required.

- Type:

  - External — Customer-facing change that affects user experience.
  - Internal — Operator-facing change that impacts internal processes.
  
- Category:

  - Configuration — Updates that require configuration adjustments.
  - Feature — Operators must update the ERS registry accordingly after the introduction of a new feature.
  - Migration — Changes that involve data, infrastructure, or version migrations.

## Creating a Notable Change

When introducing a KEB change that requires operator action, perform the following steps.

1. Create a directory for the change under [notable-changes](../../notable-changes), using the KEB release version as the directory name. For example: [notable-changes/1.22.1](../../notable-changes/1.22.1).
2. Document the change using the [Notable Change Template](../assets/notable-change-template.md). 
   1. Fill in the JSON metadata block at the top of the page.
  
      - Fields:
     
         - `requirement`: **MANDATORY** or **RECOMMENDED**
         - `type`: **EXTERNAL** or **INTERNAL**
         - `category`: **CONFIGURATION**, **FEATURE**, or **MIGRATION**
         - `additionalFiles`: number of supporting files, such as migration scripts
        
      - Example:
     
        ```json
        {
          "metadata": {
            "requirement": "RECOMMENDED",
            "type": "INTERNAL",
            "category": "CONFIGURATION",
            "additionalFiles": 0
          }
        }
        ```

   2. Clearly describe the impact, required actions, and any relevant details.
3. Within the same directory, include supporting files, such as migration scripts or configuration examples.

## Integration with Release Notes

When a directory with the corresponding release name exists, its content is automatically included in the KEB release notes. For example: [1.22.1](https://github.com/kyma-project/kyma-environment-broker/releases/tag/1.22.1)

All notable changes are also bundled into the bi-weekly KCP package.
For example, if the previous KEB version included in a KCP package was 1.21.30 and the next is 1.21.39, all notable changes from versions 1.21.31 through 1.21.39 will be included in that KCP package’s release notes.

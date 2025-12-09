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

1. Document the change using the [Notable Change Template](../assets/notable-change-template.md) in the [notable-changes-to-release](../../notable-changes-to-release) directory. 

   1. Fill in the JSON metadata block at the top of the page.
  
      - Fields:
     
         - `requirement`: **MANDATORY** or **RECOMMENDED**
         - `type`: **EXTERNAL** or **INTERNAL**
         - `category`: **CONFIGURATION**, **FEATURE**, or **MIGRATION**
        
      - Example:
     
        ```json
        {
          "metadata": {
            "requirement": "RECOMMENDED",
            "type": "INTERNAL",
            "category": "CONFIGURATION"
          }
        }
        ```

   2. Clearly describe the impact, required actions, and any relevant details.

2. Within the same directory, include supporting files, such as migration scripts or configuration examples.

## Integration with Release Notes

When the [notable-changes-to-release](../../notable-changes-to-release) directory contains at least one file, the release GitHub action creates a corresponding directory in the [notable-changes](../../notable-changes) directory for a specific KEB version release (for example, `notable-changes/1.22.1`).

All notable changes are also bundled into the bi-weekly KCP package.
For example, if the previous KEB version included in a KCP package was 1.21.30 and the next is 1.21.39, all notable changes from versions 1.21.31 through 1.21.39 will be included in that KCP package’s release notes.

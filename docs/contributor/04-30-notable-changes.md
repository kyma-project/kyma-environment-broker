# Notable changes

Notable changes refer to updates in the Kyma Environment Broker (KEB) that require operator action. These changes can be classified into two categories:
- Mandatory — Operator action is required for proper functionality.
- Optional — Operator action is recommended but not strictly required.

## Creating a Notable Change

When introducing a KEB change that requires operator action:
1. Create a directory for the change under [notable-changes](../notable-changes), using the KEB release version as the directory name.
   - Example: [notable-changes/1.22.1](../notable-changes/1.22.1)
2. Document the change using the [Notable Change Template](../assets/notable-change-template.md).
   - Clearly describe the impact, required actions, and any relevant details.
3. Include supporting files, such as migration scripts or configuration examples, within the same directory.

## Integration with Release Notes

When a directory with the corresponding release name exists, its contents will automatically be included in the [KEB release notes](https://github.com/kyma-project/kyma-environment-broker/releases).

All notable changes are also bundled into the bi-weekly KCP package.
For example, if the previous KEB version included in a KCP package was 1.21.30 and the next is 1.21.39, all notable changes from versions 1.21.31 through 1.21.39 will be included in that KCP package’s release notes.

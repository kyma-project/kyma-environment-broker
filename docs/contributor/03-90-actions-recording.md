# Actions Recording

Kyma Environment Broker (KEB) records actions as part of its audit logging and operational observability. These actions include subaccount movements and service plan updates, which are essential for tracking changes to Kyma runtimes over time.

## Overview

Actions are stored in persistent storage and are not deleted even when a runtime instance is deprovisioned. This enables historical tracking and auditing of important lifecycle events. Audit logs can be retrieved from the `/runtimes` endpoint by setting the `actions` query parameter to `true`. They are accessible through the KCP CLI and include metadata such as instance ID, timestamps, messages, action types, and old/new values.

## Supported Action Types

|     Action Type      | Description                                                                                                              |
|:--------------------:|--------------------------------------------------------------------------------------------------------------------------|
| `SubaccountMovement` | Represents the reassignment of a Kyma runtime to a different global account. See [Subaccount Movement](03-75-subaccount-movement.md). |
|     `PlanUpdate`     | Indicates a change in the service plan for a Kyma runtime. See [Service Plan Updates](03-83-plan-updates.md).                          |

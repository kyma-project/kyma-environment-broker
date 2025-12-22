# Service Plan Updates

Kyma Environment Broker (KEB) supports updating service plans. This feature allows you to change the plan of an existing Kyma runtime. However, only some plan changes are possible because the new plan must use the same provider. For example, you cannot switch from Amazon Web Services to Microsoft Azure.

> ### Note:
> For more information on recording plan updates as part of KEB's audit logging and operational observability, see [Actions](03-90-actions-recording.md).

## Configuration

To make changes to your plan, follow these steps:

1. To enable the feature, set the value: `enablePlanUpgrades: true`.
2. Define allowed plan changes in the plan configuration, for example:
```yaml
plansConfiguration:
  gcp:
    upgradableToPlans:
      - build-runtime-gcp
```

> ### Note:
> The **upgradableToPlans** field is a list of plan names to which you can upgrade the current plan. If the value is an empty (or not defined) list, or the list contains only the name of the configured plan (like `gcp` in the above example), the plan cannot be updated, and the **plan_updateable** field in the response of the `catalog` endpoint is set to `false`.

## Plan Update Request

The plan update request is similar to a regular update request. You must provide the target plan ID in the **plan_id** field. For example:

```http
PATCH /oauth/v2/service_instances/"{INSTANCE_ID}"?accepts_incomplete=true
{
    "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
    "plan_id": "{TARGET_PLAN_ID}"
}
```

When the plan update is not allowed, the response is `HTTP 400 Bad Request`.
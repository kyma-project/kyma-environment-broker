# Service Plan Updates

Kyma Environment Broker (KEB) supports updating service plans. This feature allows you to change the plan of an existing Kyma runtime. However, only some plan changes are possible because the new plan must use the same provider. For example, you cannot switch from Amazon Web Services to Microsoft Azure.

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

## Plan Update Request

The plan update request is similar to a regular update request. You must provide the target plan ID in the **plan_id** field. For example:

```http
PATCH /oauth/v2/service_instances/"{INSTANCE_ID}"?accepts_incomplete=true
{
    "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281", //Kyma ID
    "plan_id": "{TARGET_PLAN_ID}"
}
```

When the plan update is not allowed, the response is `HTTP 400 Bad Request`.
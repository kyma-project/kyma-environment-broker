# Subaccount Movement

Kyma Environment Broker (KEB) supports subaccount movement. This feature allows you to change the global account associated with a Kyma runtime without needing to deprovision and recreate the instance.

> [!NOTE]
> For details on how subaccount movement is recorded for audit purposes, see [Actions](03-90-actions.md).

## Configuration

To enable the feature, set the value of `subaccountMovementEnabled` to `true`.

## Subaccount Movement Request

The subaccount movement request is similar to a regular update request. You must provide the target global account ID in the **globalaccount_id** field. For example:

```http
PATCH /oauth/v2/service_instances/"{INSTANCE_ID}"?accepts_incomplete=true
{
   "service_id":"47c9dcbf-ff30-448e-ab36-d3bad66ba281", //Kyma ID
   "plan_id":"361c511f-f939-4621-b228-d0fb79a1fe15",
   "context":{
      "globalaccount_id":"new-globalaccount-id"
   }
}
```

If subaccount movement is not enabled, any changes to the global account ID will be ignored.

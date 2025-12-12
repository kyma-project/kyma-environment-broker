# Subaccount Movement

Kyma Environment Broker (KEB) supports subaccount movement. This feature allows you to change the global account associated with a Kyma runtime without deprovisioning and recreating the instance.

> ### Note:
> For more information on recording subaccount movement as part of KEB's audit logging and operational observability, see [Actions](03-90-actions-recording.md).

## Configuration

To enable the feature, set the value of **subaccountMovementEnabled** to `true`.

## Subaccount Movement Request

The subaccount movement request is similar to a regular update request. You must provide the target global account ID in the **globalaccount_id** field. For example:

```http
PATCH /oauth/v2/service_instances/"{INSTANCE_ID}"?accepts_incomplete=true
{
   "service_id":"47c9dcbf-ff30-448e-ab36-d3bad66ba281",
   "plan_id":"361c511f-f939-4621-b228-d0fb79a1fe15",
   "context":{
      "globalaccount_id":"new-globalaccount-id"
   }
}
```

If subaccount movement is not enabled, any changes to the global account ID are ignored.

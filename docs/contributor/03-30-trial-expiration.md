# Trial expiration

SAP BTP, Kyma runtime with Trial plan has a limited lifespan of 14 days counting from its creation time (as described in [Service description](../user/03-10-service-description.md#trial-plan)). After 14 days, [Trial Cleanup CronJob](./06-40-trial-cleanup-cronjob.md) sends a request to Kyma Environment Broker (KEB) to expire the trial instance. KEB suspends the instance without the ability to unsuspend it.

## Details

Trial Cleanup CronJob triggers the trial expiration by calling `/expire/service_instance/{instanceID}` KEB API endpoint, where `instanceID` must be a trial instance ID. The possible KEB responses are:

| Status Code | Description                                                                                                           |
| --- |-----------------------------------------------------------------------------------------------------------------------|
| 202 Accepted | Returned if the Service Instance expiration has been accepted and is in progress.                                     |
| 400 Bad Request | Returned if the request is malformed or missing mandatory data or when the instance's plan is not Trial. |
| 404 Not Found | Returned if the instance does not exist in database.                                                                  |

If KEB accepts the trial expiration request, then it marks the instance as expired by populating the instance's `ExpiredAt` field with a timestamp when request has been accepted and creates suspension operation. After suspension operation is added to the operations queue, KEB sets `parameters.ers_context.active` field to `false`. The instance is deactivated and no longer usable, can be only removed by deprovisioning request.

# Trial Expiration

SAP BTP, Kyma runtime with the trial plan has a limited lifespan of 14 days, counting from its creation time (as described in [Service Description](../user/03-10-service-description.md#trial-plan)). After 14 days, [Trial Cleanup CronJob](./06-40-trial-cleanup-cronjob.md) sends a request to Kyma Environment Broker (KEB) to expire the trial instance. KEB suspends the instance without the possibility to unsuspend it.

## Details

Trial Cleanup CronJob triggers the trial instance expiration by sending a `PUT` request to `/expire/service_instance/{instanceID}` KEB API endpoint, where `instanceID` must be a trial instance ID. The possible KEB responses are:

| Status Code | Description                                                                                             |
| --- |---------------------------------------------------------------------------------------------------------|
| 202 Accepted | Returned if the Service Instance expiration has been accepted and is in progress.                       |
| 400 Bad Request | Returned if the request is malformed, missing mandatory data, or when the instance's plan is not Trial. |
| 404 Not Found | Returned if the instance does not exist in database.                                                    |

If KEB accepts the trial instance expiration request, then it marks the instance as expired by populating the instance's `ExpiredAt` field with a timestamp when the request has been accepted. Then, it creates a suspension operation. After the suspension operation is added to the operations queue, KEB sets the **parameters.ers_context.active** field to `false`. The instance is deactivated and no longer usable. It can only be removed by deprovisioning request.

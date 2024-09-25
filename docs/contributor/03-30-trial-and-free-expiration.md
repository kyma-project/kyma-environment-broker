# Trial and Free Instance Expiration

## Overview

You can explore and use SAP BTP, Kyma runtime for free for a limited period with the following plans:
* The trial service plan for 14 days.
* The free plan for 30 days.

After the allocated time, the [Trial Cleanup](./06-40-trial-cleanup-cronjob.md) and the [Expirator](../../cmd/expirator/main.go) CronJobs send a request to Kyma Environment Broker (KEB) to expire the trial or free instance respectively. KEB suspends the instance without the possibility to unsuspend it.

## Details

The cleanup CronJob triggers the trial instance expiration by sending a `PUT` request to `/expire/service_instance/{instanceID}` KEB API endpoint, where `instanceID` must be a trial or free instance ID. The possible KEB responses are:

| Status Code | Description                                                                                             |
| --- |---------------------------------------------------------------------------------------------------------|
| 202 Accepted | Returned if the Service Instance expiration has been accepted and is in progress.                       |
| 400 Bad Request | Returned if the request is malformed, missing mandatory data, or when the instance's plan is not Trial or Free. |
| 404 Not Found | Returned if the instance does not exist in database.                                                    |

If KEB accepts the instance expiration request, then it marks the instance as expired by populating the instance's `ExpiredAt` field with a timestamp when the request is accepted. Then, it creates a suspension operation. After the suspension operation is added to the operations queue, KEB sets the **parameters.ers_context.active** field to `false`. The instance is deactivated and no longer usable. It can only be removed by deprovisioning request.

## Update Requests

When an instance update request is sent for an expired instance, the HTTP response is `200` only if the update includes a new value in the **globalaccount_id** field.
The changes to the `parameters` section are ignored.

See the example call:

```bash
PATCH /oauth/v2/service_instances/F9AC6341-AC2A-4D3E-B2B7-1A8AFAA6F4C3?accepts_incomplete=true
{
	“service_id”: “47c9dcbf-ff30-448e-ab36-d3bad66ba281", //Kyma ID
	“context”: {
		“globalaccount_id”: “{NEW_GLOBALACCOUNT_ID}”
	}
}
```

Requests that don't include a new **globalaccount_id** return the HTTP response `400`.
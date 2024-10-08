# Trial and Free Instance Expiration

## Overview

You can explore and use SAP BTP, Kyma runtime for free for a limited period with the following plans:
* The trial service plan for 14 days.
* The free plan for 30 days.

After the allocated time, the [Trial Cleanup](./06-40-trial-cleanup-cronjob.md) and the [Expirator](../../cmd/expirator/main.go) CronJobs send requests to Kyma Environment Broker (KEB) to expire the trial or free instance respectively. KEB suspends the instance without the possibility to unsuspend it.

## Details

The cleaning process is performed in the following steps:

1. The cleanup CronJobs trigger the trial or free instance expiration by sending a `PUT` request to `/expire/service_instance/{instanceID}` KEB API endpoint, where **plan** must be set to `trial` or `free`. The possible KEB responses are:

	| Status Code | Description                                                                                             |
	| --- |---------------------------------------------------------------------------------------------------------|
	| 202 Accepted | Returned if the Service Instance expiration has been accepted and is in progress.                       |
	| 400 Bad Request | Returned if the request is malformed, missing mandatory data, or when the instance's plan is not Trial or Free. |
	| 404 Not Found | Returned if the instance does not exist in the database.                                                    |

2. If KEB accepts the instance expiration request, it marks the instance as expired by populating the instance's `ExpiredAt` field with a timestamp when the request is accepted.
3. KEB creates a suspension operation, which is added to the operations queue.
4. KEB sets the **parameters.ers_context.active** field to `false`.
5. The instance is deactivated and no longer usable. It can only be removed by a deprovisioning request.

## Update Requests

When an instance update request is sent for an expired instance, the HTTP response is `200` only if the update includes a value in the **globalaccount_id** field of the `context` section.
The changes to other sections are ignored.

See the example call:

```bash
PATCH /oauth/v2/service_instances/"{INSTANCE_ID}"?accepts_incomplete=true
{
	“service_id”: “47c9dcbf-ff30-448e-ab36-d3bad66ba281", //Kyma ID
	“context”: {
		“globalaccount_id”: “{GLOBALACCOUNT_ID}”
	}
}
```

For requests that don't include **globalaccount_id** or are empty, KEB returns the HTTP response `400`.

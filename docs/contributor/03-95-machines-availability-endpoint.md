# Machines Availability Endpoint

The Machines Availability endpoint provides information about which machine-type families (for example, `m6i`, `c7i`, or `g6`) can be provisioned in specific regions, 
and whether those machine types support high availability (HA) in each region.

High availability is determined by checking how many availability zones in a given region support the specified machine type. 
If the number of zones meets or exceeds the configured threshold, the machine type is considered to be highly available for that region.

> [!NOTE]
> Currently, this endpoint supports only AWS.

## Overview

This endpoint is secured by OAuth2 token-based [authorization](01-10-authorization.md).
The list of machine types and regions originates from the Kyma Environment Broker’s (KEB) [provider configuration](02-60-plan-configuration.md). 
The broker retrieves a random hyperscaler subscription secret from Gardener and uses the associated credentials to query the cloud provider’s API. 
For each region and machine type, it determines how many availability zones support provisioning of that machine type.
If a machine type is supported in at least three availability zones within that region, it is marked as `high_availability`.

## HTTP Request

```
GET /oauth/v2/machines_availability
```
No request body is required.

## Response Structure

The response contains a list of providers. For each provider, it lists the following data:
- **machine_types** - machine-type families (for example, `m6i`, `c7i`, or `g6`)
- **regions** - supported regions for that machine type
- **high_availability** - whether enough availability zones exist in the region to maintain HA

### Response Body

```json
{
  "providers": [
    {
      "name": "AWS",
      "machine_types": [
        {
          "name": "g6",
          "regions": [
            {
              "name": "ap-south-1",
              "high_availability": false
            },
            {
              "name": "ap-southeast-2",
              "high_availability": true
            },
            {
              "name": "eu-central-1",
              "high_availability": false
            },
            {
              "name": "us-east-1",
              "high_availability": true
            },
            {
              "name": "us-west-2",
              "high_availability": true
            }
          ]
        }
      ]
    }
  ]
}
```

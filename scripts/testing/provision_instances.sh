#!/usr/bin/env bash
# Provision one or more instances via KEB API for performance or functional tests
# Usage: provision_instances.sh <count> <plan_id> <instance_name_prefix> [api_url]
# Example: provision_instances.sh 10 4deee563-e5ec-4731-b9b1-53b42d855f0c azure-cluster http://localhost:30080

set -o nounset
set -o errexit
set -E
set -o pipefail

COUNT=${1:?Number of instances required}
PLAN_ID=${2:?Plan ID required}
NAME_PREFIX=${3:?Instance name prefix required}
API_URL=${4:-http://localhost:30080}

for ((i=1;i<=COUNT;i++)); do
  uid=$(uuidgen)
  curl --request PUT \
    --url "$API_URL/oauth/v2/service_instances/$uid" \
    --header "Content-Type: application/json" \
    --header "X-Broker-API-Version: 2.16" \
    --data '{
      "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
      "plan_id": "'$PLAN_ID'",
      "context": {
        "globalaccount_id": "2f5011af-2fd3-44ba-ac60-eeb1148c2995",
        "subaccount_id": "8b9a0db4-9aef-4da2-a856-61a4420b66fd",
        "user_id": "user@email.com"
      },
      "parameters": {
        "name": "'$NAME_PREFIX'-'$i'",
        "region": "northeurope"
      }
    }'
done

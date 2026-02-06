#!/usr/bin/env bash
# Update a single instance with parameters
# Usage: update_instance.sh <instance_id> <plan_id> <parameter_json> [base_url]
# Example: update_instance.sh azure-cluster 4deee563-e5ec-4731-b9b1-53b42d855f0c '{"machineType":"Standard_D4s_v5"}' http://localhost:8080

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

INSTANCE_ID=${1:?Instance ID required}
PLAN_ID=${2:?Plan ID required}
PARAMETER_JSON=${3:?Parameter JSON required}
BASE_URL=${4:-http://localhost:8080}
GLOBAL_ACCOUNT_ID=${GLOBAL_ACCOUNT_ID:-2f5011af-2fd3-44ba-ac60-eeb1148c2995}

curl --request PATCH \
  --url "${BASE_URL}/oauth/v2/service_instances/${INSTANCE_ID}?accepts_incomplete=true" \
  --header 'Content-Type: application/json' \
  --header 'X-Broker-API-Version: 2.16' \
  --data "{\"service_id\":\"47c9dcbf-ff30-448e-ab36-d3bad66ba281\",\"plan_id\":\"${PLAN_ID}\",\"context\":{\"globalaccount_id\":\"${GLOBAL_ACCOUNT_ID}\",\"subaccount_id\":\"8b9a0db4-9aef-4da2-a856-61a4420b66fd\",\"user_id\":\"user@email.com\"},\"parameters\":${PARAMETER_JSON}}"

echo "Update request sent for instance ${INSTANCE_ID}"

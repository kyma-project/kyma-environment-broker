#!/usr/bin/env bash
# Deprovision instance(s) - either a single instance by ID or all instances of a plan
# Usage: deprovision_instances.sh <plan_id> <plan_name_or_instance_id> [base_url] [single]
#   For bulk: deprovision_instances.sh <plan_id> <plan_name> [base_url]
#   For single: deprovision_instances.sh <plan_id> <instance_id> [base_url] single

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

PLAN_ID=${1:?Plan ID required}
PLAN_NAME_OR_INSTANCE=${2:?Plan name or instance ID required}
BASE_URL=${3:-http://localhost:30080}
MODE=${4:-bulk}
GLOBAL_ACCOUNT_ID=${GLOBAL_ACCOUNT_ID:-2f5011af-2fd3-44ba-ac60-eeb1148c2995}

# Single instance mode
if [ "$MODE" = "single" ]; then
  INSTANCE_ID="$PLAN_NAME_OR_INSTANCE"
  curl --request DELETE \
    --url "${BASE_URL}/oauth/v2/service_instances/${INSTANCE_ID}?accepts_incomplete=true&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281&plan_id=${PLAN_ID}" \
    --header 'Content-Type: application/json' \
    --header 'X-Broker-API-Version: 2.16' \
    --data "{\"service_id\":\"47c9dcbf-ff30-448e-ab36-d3bad66ba281\",\"plan_id\":\"${PLAN_ID}\",\"context\":{\"globalaccount_id\":\"${GLOBAL_ACCOUNT_ID}\",\"subaccount_id\":\"8b9a0db4-9aef-4da2-a856-61a4420b66fd\",\"user_id\":\"user@email.com\"}}"
  echo "Deprovision request sent for instance ${INSTANCE_ID}"
  exit 0
fi

# Bulk mode
PLAN_NAME="$PLAN_NAME_OR_INSTANCE"
PAGE=1

while true; do
  RESPONSE=$(curl --request GET \
    --url "${BASE_URL}/runtimes?plan=${PLAN_NAME}&page=${PAGE}" \
    --header 'Content-Type: application/json' \
    --header 'X-Broker-API-Version: 2.16')

  COUNT=$(echo "$RESPONSE" | jq '.count')
  if [ "$COUNT" -eq 0 ]; then
    break
  fi

  echo "$RESPONSE" | jq -r '.data[].instanceID' | while read -r INSTANCE_ID; do
    curl --request DELETE \
      --url "${BASE_URL}/oauth/v2/service_instances/$INSTANCE_ID?accepts_incomplete=true&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281&plan_id=${PLAN_ID}" \
      --header "Content-Type: application/json" \
      --header "X-Broker-API-Version: 2.16" \
      --data "{\"service_id\":\"47c9dcbf-ff30-448e-ab36-d3bad66ba281\",\"plan_id\":\"${PLAN_ID}\",\"context\":{\"globalaccount_id\":\"2f5011af-2fd3-44ba-ac60-eeb1148c2995\",\"subaccount_id\":\"8b9a0db4-9aef-4da2-a856-61a4420b66fd\",\"user_id\":\"user@email.com\"}}"
  done

  PAGE=$((PAGE + 1))
done

echo "Deprovision requests sent for all instances of plan ${PLAN_NAME}"

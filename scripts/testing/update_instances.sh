#!/usr/bin/env bash
# Update all instances of a given plan with a parameter
# Usage: update_instances.sh <plan_id> <plan_name> <parameter_name> <parameter_value> [base_url]

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

PLAN_ID=${1:?Plan ID required}
PLAN_NAME=${2:?Plan name required}
PARAMETER_NAME=${3:?Parameter name required}
PARAMETER_VALUE=${4:?Parameter value required}
BASE_URL=${5:-http://localhost:30080}

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
    # Check if parameter value is numeric or string and format accordingly
    if [[ "$PARAMETER_VALUE" =~ ^[0-9]+$ ]]; then
      curl --request PATCH \
        --url "${BASE_URL}/oauth/v2/service_instances/$INSTANCE_ID?accepts_incomplete=true" \
        --header "Content-Type: application/json" \
        --header "X-Broker-API-Version: 2.16" \
        --data "{\"service_id\":\"47c9dcbf-ff30-448e-ab36-d3bad66ba281\",\"plan_id\":\"${PLAN_ID}\",\"context\":{\"globalaccount_id\":\"2f5011af-2fd3-44ba-ac60-eeb1148c2995\",\"subaccount_id\":\"8b9a0db4-9aef-4da2-a856-61a4420b66fd\",\"user_id\":\"user@email.com\"},\"parameters\":{\"${PARAMETER_NAME}\":${PARAMETER_VALUE}}}"
    else
      curl --request PATCH \
        --url "${BASE_URL}/oauth/v2/service_instances/$INSTANCE_ID?accepts_incomplete=true" \
        --header "Content-Type: application/json" \
        --header "X-Broker-API-Version: 2.16" \
        --data "{\"service_id\":\"47c9dcbf-ff30-448e-ab36-d3bad66ba281\",\"plan_id\":\"${PLAN_ID}\",\"context\":{\"globalaccount_id\":\"2f5011af-2fd3-44ba-ac60-eeb1148c2995\",\"subaccount_id\":\"8b9a0db4-9aef-4da2-a856-61a4420b66fd\",\"user_id\":\"user@email.com\"},\"parameters\":{\"${PARAMETER_NAME}\":\"${PARAMETER_VALUE}\"}}"
    fi
  done

  PAGE=$((PAGE + 1))
done

echo "Update requests sent for all instances of plan ${PLAN_NAME}"

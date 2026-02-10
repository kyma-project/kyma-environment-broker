#!/usr/bin/env bash

set -o nounset
set -o errexit
set -E
set -o pipefail

INSTANCE_ID=${1:?Instance ID required}
PLAN_ID=${2:?Plan ID required}
UPDATES_NUMBER=${3:?Number of updates required}
KEB_HOST=${4:-http://localhost:8080}

echo "Updating instance $INSTANCE_ID $UPDATES_NUMBER times..."

max=5
for i in $(seq 1 "$UPDATES_NUMBER"); do
  curl --request PATCH \
    --url "$KEB_HOST/oauth/v2/service_instances/$INSTANCE_ID?accepts_incomplete=true" \
    --header "Content-Type: application/json" \
    --header "X-Broker-API-Version: 2.16" \
    --data "{\"service_id\":\"47c9dcbf-ff30-448e-ab36-d3bad66ba281\",\"plan_id\":\"$PLAN_ID\",\"context\":{\"globalaccount_id\":\"2f5011af-2fd3-44ba-ac60-eeb1148c2995\",\"subaccount_id\":\"8b9a0db4-9aef-4da2-a856-61a4420b66fd\",\"user_id\":\"user@email.com\"},\"parameters\":{\"autoScalerMax\":$max}}"
  
  max=$((max + 1))
  if [ "$max" -gt 100 ]; then
    max=5
  fi
done

echo "Completed $UPDATES_NUMBER updates for instance $INSTANCE_ID"

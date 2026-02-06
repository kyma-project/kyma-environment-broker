#!/usr/bin/env bash
# Provision instance(s) via KEB API for performance or functional tests
# 
# Single instance (count=1):
#   provision_instances.sh 1 <instance_id> <plan_id> <name> <region> [base_url]
#   Example: provision_instances.sh 1 azure-cluster 4deee563-e5ec-4731-b9b1-53b42d855f0c azure-cluster northeurope http://localhost:30080
#
# Multiple instances (count>1):
#   provision_instances.sh <count> <plan_id> <instance_name_prefix> <region> [api_url]
#   Example: provision_instances.sh 10 4deee563-e5ec-4731-b9b1-53b42d855f0c azure-cluster northeurope http://localhost:30080

set -o nounset
set -o errexit
set -E
set -o pipefail

COUNT=${1:?Count required}
GLOBAL_ACCOUNT_ID=${GLOBAL_ACCOUNT_ID:-2f5011af-2fd3-44ba-ac60-eeb1148c2995}

if [[ "$COUNT" == "1" ]]; then
  # Single instance
  INSTANCE_ID=${2:?Instance ID required}
  PLAN_ID=${3:?Plan ID required}
  NAME=${4:?Name required}
  REGION=${5:?Region required}
  BASE_URL=${6:-http://localhost:30080}
  
  curl --request PUT \
    --url "${BASE_URL}/oauth/v2/service_instances/${INSTANCE_ID}" \
    --header "Content-Type: application/json" \
    --header "X-Broker-API-Version: 2.16" \
    --data "{\"service_id\":\"47c9dcbf-ff30-448e-ab36-d3bad66ba281\",\"plan_id\":\"${PLAN_ID}\",\"context\":{\"globalaccount_id\":\"${GLOBAL_ACCOUNT_ID}\",\"subaccount_id\":\"8b9a0db4-9aef-4da2-a856-61a4420b66fd\",\"user_id\":\"user@email.com\"},\"parameters\":{\"name\":\"${NAME}\",\"region\":\"${REGION}\"}}"
  
  echo "Provision request sent for instance ${INSTANCE_ID}"
else
  # Multiple instances
  PLAN_ID=${2:?Plan ID required}
  NAME_PREFIX=${3:?Instance name prefix required}
  REGION=${4:?Region required}
  API_URL=${5:-http://localhost:30080}
  
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
          "region": "'$REGION'"
        }
      }'
  done
fi

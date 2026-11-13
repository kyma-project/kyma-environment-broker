#!/usr/bin/env bash
# Deprovision all instances known to KEB.
# Usage: ./scripts/cleanup-all-runtimes.sh [base_url]

set -o nounset
set -o errexit
set -E
set -o pipefail

BASE_URL="${1:-http://localhost:8080}"

# ─── Safety check: refuse to run against anything other than a local k3d cluster ─

CURRENT_CONTEXT=$(kubectl config current-context 2>/dev/null || true)
if [[ "$CURRENT_CONTEXT" != k3d-* ]]; then
  echo "ERROR: current kubectl context is '${CURRENT_CONTEXT}', which does not look like a local k3d cluster."
  echo "This script deprovisions ALL instances and must only run against a local k3d cluster."
  exit 1
fi

SERVICE_ID="47c9dcbf-ff30-448e-ab36-d3bad66ba281"
GLOBAL_ACCOUNT_ID="2f5011af-2fd3-44ba-ac60-eeb1148c2995"
SUBACCOUNT_ID="8b9a0db4-9aef-4da2-a856-61a4420b66fd"
POLL_INTERVAL=5

log() { echo "[$(date '+%H:%M:%S')] $*"; }

# ─── Collect all instance IDs ─────────────────────────────────────────────────

log "Fetching all instances from ${BASE_URL}/runtimes ..."

INSTANCE_IDS=()
PAGE=1
while true; do
  RESPONSE=$(curl --silent --fail "${BASE_URL}/runtimes?page=${PAGE}&pageSize=100")
  COUNT=$(echo "$RESPONSE" | jq '.count')
  if [[ "$COUNT" -eq 0 ]]; then
    break
  fi
  while IFS= read -r IID; do
    INSTANCE_IDS+=("$IID")
  done < <(echo "$RESPONSE" | jq -r '.data[].instanceID')
  PAGE=$((PAGE + 1))
done

TOTAL=${#INSTANCE_IDS[@]}
if [[ $TOTAL -eq 0 ]]; then
  log "No instances found, nothing to do."
  exit 0
fi

log "Found ${TOTAL} instance(s) to deprovision."

# ─── Send deprovision requests ────────────────────────────────────────────────

for INSTANCE_ID in "${INSTANCE_IDS[@]}"; do
  PLAN_ID=$(curl --silent "${BASE_URL}/runtimes" | \
    jq -r --arg iid "$INSTANCE_ID" '.data[] | select(.instanceID==$iid) | .servicePlanID')

  curl --silent --output /dev/null --request DELETE \
    --url "${BASE_URL}/oauth/v2/service_instances/${INSTANCE_ID}?accepts_incomplete=true&service_id=${SERVICE_ID}&plan_id=${PLAN_ID}" \
    --header "Content-Type: application/json" \
    --header "X-Broker-API-Version: 2.16" \
    --data "{
      \"service_id\": \"${SERVICE_ID}\",
      \"plan_id\": \"${PLAN_ID}\",
      \"context\": {
        \"globalaccount_id\": \"${GLOBAL_ACCOUNT_ID}\",
        \"subaccount_id\": \"${SUBACCOUNT_ID}\",
        \"user_id\": \"user@email.com\"
      }
    }" || true
  log "  Deprovision request sent: ${INSTANCE_ID}"
done

# ─── Wait for all to disappear ────────────────────────────────────────────────

log "Waiting for all instances to be deprovisioned..."

while true; do
  RESPONSE=$(curl --silent "${BASE_URL}/runtimes?pageSize=100")
  REMAINING=$(echo "$RESPONSE" | jq '.totalCount')
  if [[ "$REMAINING" -eq 0 ]]; then
    log "All instances deprovisioned."
    break
  fi
  log "  ${REMAINING} instance(s) still remaining..."
  sleep "$POLL_INTERVAL"
done

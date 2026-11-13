#!/usr/bin/env bash
# Performance test: provision N runtimes, wait for all to succeed,
# then deprovision DEPROV_BATCH at the same time, wait for all to succeed.
#
# Usage: ./scripts/perf-test.sh [N] [DEPROV_BATCH]
#   N            - total instances to provision (default: 20)
#   DEPROV_BATCH - how many to deprovision concurrently (default: 10)
#
# Prerequisites:
#   - KEB running and port-forwarded on localhost:8080
#   - kubectl configured to kcp-system namespace
#   - jq installed

set -o nounset
set -o errexit
set -E
set -o pipefail

N=${1:-20}
DEPROV_BATCH=${2:-10}
BASE_URL="http://localhost:8080"
PLAN_ID="4deee563-e5ec-4731-b9b1-53b42d855f0c"  # azure
SERVICE_ID="47c9dcbf-ff30-448e-ab36-d3bad66ba281"
GLOBAL_ACCOUNT_ID="2f5011af-2fd3-44ba-ac60-eeb1148c2995"
SUBACCOUNT_ID="8b9a0db4-9aef-4da2-a856-61a4420b66fd"
POLL_INTERVAL=2

INSTANCE_IDS=()
PREFIX=$(cat /dev/urandom | LC_ALL=C tr -dc 'a-z0-9' | dd bs=1 count=4 2>/dev/null || true)

log() { echo "[$(date '+%H:%M:%S')] $*"; }

# ─── Pre-flight check ─────────────────────────────────────────────────────────
if ! curl --silent --fail --max-time 2 "${BASE_URL}/runtimes" &>/dev/null; then
  echo "ERROR: KEB not reachable at ${BASE_URL}. Run:"
  echo "  kubectl port-forward -n kcp-system deployment/kcp-kyma-environment-broker 8080:8080 &"
  exit 1
fi

# ─── STEP 1: Provision N instances ───────────────────────────────────────────

log "=== STEP 1: Provisioning ${N} instances ==="

for ((i = 1; i <= N; i++)); do
  INSTANCE_ID="${PREFIX}-perf-$(printf '%04d' "$i")"
  INSTANCE_IDS+=("$INSTANCE_ID")

  curl --silent --output /dev/null --request PUT \
    --url "${BASE_URL}/oauth/v2/service_instances/${INSTANCE_ID}?accepts_incomplete=true" \
    --header "Content-Type: application/json" \
    --header "X-Broker-API-Version: 2.16" \
    --data "{
      \"service_id\": \"${SERVICE_ID}\",
      \"plan_id\": \"${PLAN_ID}\",
      \"context\": {
        \"globalaccount_id\": \"${GLOBAL_ACCOUNT_ID}\",
        \"subaccount_id\": \"${SUBACCOUNT_ID}\",
        \"user_id\": \"user@email.com\",
        \"sm_operator_credentials\": {
          \"clientid\": \"cid\", \"clientsecret\": \"cs\",
          \"url\": \"url\", \"sm_url\": \"sm_url\"
        }
      },
      \"parameters\": {\"name\": \"${INSTANCE_ID}\", \"region\": \"northeurope\"}
    }"

  log "  Provision request sent: ${INSTANCE_ID}"
done

# ─── STEP 2: Simulate KIM + KLM, wait for all provisioning to succeed ────────

# Build a kubeconfig pointing to the local k3d cluster (reachable from inside the pod)
LOCAL_KUBECONFIG=$(kubectl config view --minify --raw \
  | sed "s|https://0\.0\.0\.0|https://$(ifconfig en0 2>/dev/null | awk '$1=="inet" {print $2}' || ifconfig eth0 2>/dev/null | awk '$1=="inet" {print $2}')|" \
  | sed "s|https://127\.0\.0\.1|https://$(ifconfig en0 2>/dev/null | awk '$1=="inet" {print $2}' || ifconfig eth0 2>/dev/null | awk '$1=="inet" {print $2}')|" \
  | yq 'del(.clusters[].cluster."certificate-authority-data") | .clusters[].cluster."insecure-skip-tls-verify" = true')

log "=== STEP 2: Simulating KIM and KLM ==="

PROVISION_TIMEOUT=${PROVISION_TIMEOUT:-300}  # seconds, override with env var
PROVISION_DEADLINE=$(( $(date +%s) + PROVISION_TIMEOUT ))

while true; do
  # Patch all Runtime CRs not yet Ready (including those with no status yet)
  while IFS= read -r RUNTIME_ID; do
    [[ -z "$RUNTIME_ID" ]] && continue

    # Create kubeconfig secret pointing to local k3d cluster
    kubectl create secret generic "kubeconfig-${RUNTIME_ID}" \
      -n kcp-system \
      --from-literal=config="${LOCAL_KUBECONFIG}" \
      --dry-run=client -o yaml | kubectl apply -f - &>/dev/null || true

    log "  Patching Runtime ${RUNTIME_ID} -> Ready"
    kubectl patch runtime "${RUNTIME_ID}" -n kcp-system \
      --type merge --subresource status \
      -p '{"status": {"state": "Ready"}}' &>/dev/null || true
  done < <(kubectl get runtime -n kcp-system -o json | \
    jq -r '.items[] | select((.status.state // "") != "Ready") | .metadata.name' 2>/dev/null || true)

  # Patch all Kyma CRs not yet Ready
  while IFS= read -r KYMA_ID; do
    [[ -z "$KYMA_ID" ]] && continue
    log "  Patching Kyma ${KYMA_ID} -> Ready"
    kubectl patch kyma "${KYMA_ID}" -n kcp-system \
      --type merge --subresource status \
      -p '{"status": {"state": "Ready"}}' &>/dev/null || true
  done < <(kubectl get kyma -n kcp-system -o json | \
    jq -r '.items[] | select(.status.state != "Ready") | .metadata.name' 2>/dev/null || true)

  # Single runtimes call — check all instances at once
  RUNTIMES_JSON=$(curl --silent --request GET \
    --url "${BASE_URL}/runtimes?pageSize=100" \
    --header "X-Broker-API-Version: 2.16")

  PENDING=0
  FAILED=0
  for INSTANCE_ID in "${INSTANCE_IDS[@]}"; do
    STATE=$(echo "$RUNTIMES_JSON" | jq -r --arg iid "$INSTANCE_ID" \
      '.data[] | select(.instanceID==$iid) | .status.provisioning.state' 2>/dev/null || echo "")
    if [[ "$STATE" == "failed" ]]; then
      FAILED=$((FAILED + 1))
    elif [[ "$STATE" != "succeeded" ]]; then
      PENDING=$((PENDING + 1))
    fi
  done

  if [[ $FAILED -gt 0 ]]; then
    log "ERROR: ${FAILED} provisioning(s) failed"
    exit 1
  fi

  log "  Provisioning: ${PENDING} pending, $((N - PENDING)) succeeded"

  if [[ $PENDING -eq 0 ]]; then
    log "  All ${N} instances provisioned successfully."
    break
  fi

  if [[ $(date +%s) -ge $PROVISION_DEADLINE ]]; then
    log "ERROR: Provisioning timed out after ${PROVISION_TIMEOUT}s — ${PENDING} instance(s) still pending"
    exit 1
  fi

  sleep "$POLL_INTERVAL"
done

# ─── STEP 3: Create Shoot for each instance ───────────────────────────────────

log "=== STEP 3: Creating Shoot resources ==="

for INSTANCE_ID in "${INSTANCE_IDS[@]}"; do
  RUNTIME_ID=$(kubectl get runtime -n kcp-system -o json | \
    jq -r --arg iid "$INSTANCE_ID" \
      '.items[] | select(.metadata.labels["kyma-project.io/instance-id"]==$iid) | .metadata.name' \
    2>/dev/null || true)

  if [[ -z "$RUNTIME_ID" ]]; then
    log "  WARN: Could not find Runtime for ${INSTANCE_ID}, skipping shoot creation"
    continue
  fi

  ./scripts/create_shoot.sh "$RUNTIME_ID" &>/dev/null || true
  log "  Shoot created for Runtime ${RUNTIME_ID} (instance ${INSTANCE_ID})"
done

# ─── STEP 4: Deprovision DEPROV_BATCH instances concurrently ──────────────────

log "=== STEP 4: Deprovisioning ${DEPROV_BATCH} instances concurrently ==="

DEPROV_INSTANCE_IDS=("${INSTANCE_IDS[@]:0:$DEPROV_BATCH}")

for INSTANCE_ID in "${DEPROV_INSTANCE_IDS[@]}"; do
  curl --silent --output /dev/null --request DELETE \
    "--url" "${BASE_URL}/oauth/v2/service_instances/${INSTANCE_ID}?accepts_incomplete=true&service_id=${SERVICE_ID}&plan_id=${PLAN_ID}" \
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
    }"
  log "  Deprovision request sent: ${INSTANCE_ID}"
done

# ─── STEP 5: Wait for all deprovisionings to finish ───────────────────────────

log "=== STEP 5: Waiting for ${DEPROV_BATCH} deprovisionings to complete ==="

while true; do
  RUNTIMES_JSON=$(curl --silent --request GET \
    --url "${BASE_URL}/runtimes?pageSize=100" \
    --header "X-Broker-API-Version: 2.16")

  PENDING=0
  FAILED=0
  for INSTANCE_ID in "${DEPROV_INSTANCE_IDS[@]}"; do
    STATE=$(echo "$RUNTIMES_JSON" | jq -r --arg iid "$INSTANCE_ID" \
      '.data[] | select(.instanceID==$iid) | .status.deprovisioning.state' 2>/dev/null || echo "")
    if [[ "$STATE" == "failed" ]]; then
      FAILED=$((FAILED + 1))
    elif [[ -n "$(echo "$RUNTIMES_JSON" | jq -r --arg iid "$INSTANCE_ID" '.data[] | select(.instanceID==$iid) | .instanceID' 2>/dev/null || true)" ]]; then
      PENDING=$((PENDING + 1))
    fi
  done

  if [[ $FAILED -gt 0 ]]; then
    log "ERROR: ${FAILED} deprovisioning(s) failed"
    exit 1
  fi

  log "  Deprovisioning: ${PENDING} pending, $((DEPROV_BATCH - PENDING)) done"

  if [[ $PENDING -eq 0 ]]; then
    log "  All ${DEPROV_BATCH} deprovisionings completed successfully."
    break
  fi

  sleep "$POLL_INTERVAL"
done

log "=== DONE ==="
log "Provisioned: ${N}, Deprovisioned concurrently: ${DEPROV_BATCH}"

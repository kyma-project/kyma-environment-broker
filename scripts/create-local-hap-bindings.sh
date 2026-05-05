#!/bin/bash
# Creates dummy CredentialsBinding CRs in garden-kyma-dev to allow local seeding
# of 1000+ instances without exhausting the HAP pool.
#
# Each hyperscaler binding holds up to 3 instances (the local aws limit).
# We create enough bindings per hyperscaler to cover the desired instance count.
#
# Usage: ./scripts/create-local-hap-bindings.sh [--instances N]
#   --instances N   Total instances to support (default: 1000)

set -o nounset
set -o errexit
set -o pipefail

INSTANCES=${INSTANCES:-1000}
LIMIT=3  # must match hap.multiHyperscalerAccount.limits.aws in scripts/values.yaml

while [[ $# -gt 0 ]]; do
  case $1 in
    --instances) INSTANCES="$2"; shift 2 ;;
    *) echo "Unknown argument: $1"; exit 1 ;;
  esac
done

# Plan weight distribution from seed_analytics.py (approximate)
# aws=45%, azure=33%, gcp=14%, azure_lite=5%, trial=3%
AWS_INSTANCES=$(( (INSTANCES * 45 / 100) + LIMIT ))
AZURE_INSTANCES=$(( (INSTANCES * 38 / 100) + LIMIT ))  # azure + azure_lite
GCP_INSTANCES=$(( (INSTANCES * 14 / 100) + LIMIT ))

AWS_BINDINGS=$(( (AWS_INSTANCES + LIMIT - 1) / LIMIT ))
AZURE_BINDINGS=$(( (AZURE_INSTANCES + LIMIT - 1) / LIMIT ))
GCP_BINDINGS=$(( (GCP_INSTANCES + LIMIT - 1) / LIMIT ))

echo "Creating CredentialsBindings for ~${INSTANCES} instances (limit=${LIMIT} per binding):"
echo "  AWS:   ${AWS_BINDINGS} bindings"
echo "  Azure: ${AZURE_BINDINGS} bindings"
echo "  GCP:   ${GCP_BINDINGS} bindings"

create_bindings() {
  local type=$1
  local secret=$2
  local count=$3

  for i in $(seq 1 "$count"); do
    local name="${type}-seed-${i}"
    kubectl apply -f - -n garden-kyma-dev <<EOF
apiVersion: security.gardener.cloud/v1alpha1
kind: CredentialsBinding
metadata:
  labels:
    hyperscalerType: ${type}
  name: ${name}
  namespace: garden-kyma-dev
provider:
  type: ${type}
credentialsRef:
  name: ${secret}
  namespace: garden-kyma-dev
EOF
  done
  echo "  Created ${count} ${type} bindings."
}

create_bindings "aws"   "aws-secret"   "$AWS_BINDINGS"
create_bindings "azure" "azure-secret" "$AZURE_BINDINGS"
create_bindings "gcp"   "gcp-secret"   "$GCP_BINDINGS"

echo "Done. Run 'kubectl get credentialsbinding -n garden-kyma-dev' to verify."

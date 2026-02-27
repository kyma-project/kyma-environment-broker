#!/bin/bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

RUNTIME_ID=${1:-}

if [ -z "$RUNTIME_ID" ]; then
  echo "Usage: $0 <runtime-id>"
  echo "Creates a minimal Shoot resource for the given runtime ID using values from the Runtime CR"
  exit 1
fi

# Get runtime resource and extract needed fields
RUNTIME_JSON=$(kubectl get runtime -n kcp-system "$RUNTIME_ID" -o json)

ACCOUNT_ID=$(echo "$RUNTIME_JSON" | jq -r '.metadata.labels["kyma-project.io/global-account-id"]')
CREDENTIALS_BINDING=$(echo "$RUNTIME_JSON" | jq -r '.spec.shoot.secretBindingName')

if [ -z "$ACCOUNT_ID" ] || [ "$ACCOUNT_ID" = "null" ]; then
  echo "Error: Could not extract global account ID from runtime $RUNTIME_ID"
  exit 1
fi

if [ -z "$CREDENTIALS_BINDING" ] || [ "$CREDENTIALS_BINDING" = "null" ]; then
  echo "Error: Could not extract secretBindingName from runtime $RUNTIME_ID"
  exit 1
fi

cat <<EOF | kubectl apply -f -
apiVersion: core.gardener.cloud/v1beta1
kind: Shoot
metadata:
  name: ${RUNTIME_ID}
  namespace: garden-kyma-dev
  labels:
    account: ${ACCOUNT_ID}
spec:
  cloudProfileName: not-implemented
  region: not-implemented
  credentialsBindingName: ${CREDENTIALS_BINDING}
  purpose: testing
EOF

echo "Shoot '${RUNTIME_ID}' created successfully with account '${ACCOUNT_ID}' and credentials binding '${CREDENTIALS_BINDING}'"

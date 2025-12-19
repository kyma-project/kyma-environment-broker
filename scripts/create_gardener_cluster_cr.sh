#!/bin/bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

GLOBAL_ACCOUNT_ID=$1
RUNTIME_ID=$2

if [ -z "$GLOBAL_ACCOUNT_ID" ] || [ -z "$RUNTIME_ID" ]; then
  echo "Usage: $0 <global-account-id> <runtime-id>"
  exit 1
fi

echo "Creating GardenerCluster '$RUNTIME_ID' with global account ID '$GLOBAL_ACCOUNT_ID' in namespace kcp-system..."

cat <<EOF | kubectl apply -f -
apiVersion: infrastructuremanager.kyma-project.io/v1
kind: GardenerCluster
metadata:
  name: $RUNTIME_ID
  namespace: kcp-system
  labels:
    kyma-project.io/global-account-id: $GLOBAL_ACCOUNT_ID
spec:
  kubeconfig:
    secret:
      key: config
      name: kubeconfig-test
      namespace: kcp-system
  shoot:
    name: test-shoot
EOF

echo "GardenerCluster created successfully."

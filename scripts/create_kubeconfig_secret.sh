#!/bin/bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

RUNTIME_ID=$1

if [ -z "$RUNTIME_ID" ]; then
  echo "Usage: $0 <runtime-id>"
  echo "Creates a kubeconfig secret for the given runtime ID to simulate KIM"
  exit 1
fi

echo "Creating kubeconfig secret for runtime '$RUNTIME_ID' in namespace kcp-system..."

# In test environment, runtime cluster is the same as KCP cluster
KUBE_SERVER_IP=$(ifconfig en0 | awk '$1=="inet" {print $2}' || ifconfig eth0 | awk '$1=="inet" {print $2}')
KCFG=$(kubectl config view --minify --raw \
      | sed "s|https://0\.0\.0\.0|https://${KUBE_SERVER_IP}|" \
      | sed "s|https://127\.0\.0\.1|https://${KUBE_SERVER_IP}|" \
       | yq 'del(.clusters[].cluster."certificate-authority-data") | .clusters[].cluster."insecure-skip-tls-verify" = true')

# Create the secret with the kubeconfig
kubectl create secret generic "kubeconfig-${RUNTIME_ID}" \
  --from-literal=config="$KCFG" \
  -n kcp-system \
  --dry-run=client -o yaml | kubectl apply -f -

if [ $? -eq 0 ]; then
  echo "Successfully created kubeconfig secret for runtime '$RUNTIME_ID'"
else
  echo "Failed to create kubeconfig secret"
  exit 1
fi

#!/bin/bash

set -o nounset
set -o errexit
set -o pipefail

CLUSTER_NAME="${1:-kyma}"

echo "Deleting k3d cluster '${CLUSTER_NAME}' (if exists)..."
k3d cluster delete "${CLUSTER_NAME}" 2>/dev/null || true

echo "Creating k3d cluster '${CLUSTER_NAME}'..."
k3d cluster create "${CLUSTER_NAME}" --k3s-arg "--tls-san=0.0.0.0@server:0"

echo "Cluster '${CLUSTER_NAME}' is ready."
kubectl cluster-info

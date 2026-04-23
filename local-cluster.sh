#!/usr/bin/env bash
# Recreates the local k3d cluster (no local registry needed).
# Not tracked by git.

set -o nounset
set -o errexit
set -o pipefail

echo "Deleting existing 'kyma' cluster (if any)..."
k3d cluster delete kyma 2>/dev/null || true

echo "Creating k3d cluster 'kyma'..."
k3d cluster create kyma \
  --port "8080:80@loadbalancer" \
  --wait

echo "Cluster is ready."
echo "  kubectl: $(kubectl config current-context)"

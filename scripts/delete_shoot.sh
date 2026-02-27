#!/bin/bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

RUNTIME_ID=${1:-}

if [ -z "$RUNTIME_ID" ]; then
    echo "Usage: $0 <RUNTIME_ID>"
    exit 1
fi

echo "Deleting Shoot '$RUNTIME_ID' in namespace 'garden-kyma-dev'..."
if kubectl delete shoot "$RUNTIME_ID" -n garden-kyma-dev; then
    echo "Shoot '$RUNTIME_ID' deleted"
else
    echo "Failed to delete Shoot '$RUNTIME_ID' (it may not exist)" >&2
    exit 1
fi

#!/bin/bash

set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.

cd "$(dirname "$0")" || exit

LOCAL_BIN=$(pwd)/bin/$$
mkdir -p "$LOCAL_BIN"

K8S_VERSION=1.29.1

GOBIN="$LOCAL_BIN" go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

output=$("$LOCAL_BIN"/setup-envtest use --bin-dir "$LOCAL_BIN" -p path "$K8S_VERSION")
echo "$output"
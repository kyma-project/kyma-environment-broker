#!/bin/bash
cd "$(dirname "$0")" || exit

LOCAL_BIN=$(pwd)/bin/$$
mkdir -p "$LOCAL_BIN"

K8S_VERSION=1.29.1

GOBIN="$LOCAL_BIN" go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

output=$("$LOCAL_BIN"/setup-envtest use --bin-dir "$LOCAL_BIN" -p path "$K8S_VERSION")
if [ $? -ne 0 ]; then
  echo "Error: failed to run setup-envtest"
  exit $?
fi
echo "$output"
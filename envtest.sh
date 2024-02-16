#!/bin/bash
cd "$(dirname "$0")" || exit

LOCAL_BIN=$(pwd)/bin
mkdir -p "$LOCAL_BIN"

K8S_VERSION=1.29.1

#check if setup-envtest is installed or check if currently installed setup-envtest contains assets for requested k8s version
if [ ! -e "$LOCAL_BIN/setup-envtest" ] || [ -z $("$LOCAL_BIN"/setup-envtest list -i  | awk '{print $2}' | tr -d v | grep "$K8S_VERSION") ]; then
  GOBIN="$LOCAL_BIN" go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
  if [ $? -ne 0 ]; then
    echo "Error: failed to install setup-envtest"
    exit $?
  fi
fi

output=$("$LOCAL_BIN"/setup-envtest use --bin-dir "$LOCAL_BIN" -p path "$K8S_VERSION")
if [ $? -ne 0 ]; then
  echo "Error: failed to run setup-envtest"
  exit $?
fi
echo "$output"
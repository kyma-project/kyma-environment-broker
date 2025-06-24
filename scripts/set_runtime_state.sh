#!/bin/bash

RUNTIME_ID=$1
STATE=$2

if [ -z "$RUNTIME_ID" ] || [ -z "$STATE" ]; then
  echo "Usage: $0 <runtime-id> <state>"
  echo "Valid states: Pending, Ready, Terminating, Failed"
  exit 1
fi

case "$STATE" in
  Pending|Ready|Terminating|Failed)
    ;;
  *)
    echo "Invalid state: $STATE"
    echo "Valid states: Pending, Ready, Terminating, Failed"
    exit 1
    ;;
esac

echo "Patching Runtime '$RUNTIME_ID' in namespace kcp-system to state '$STATE'..."

kubectl patch runtime "$RUNTIME_ID" \
  -n kcp-system \
  --type merge \
  --subresource status \
  -p "{\"status\": {\"state\": \"$STATE\"}}"

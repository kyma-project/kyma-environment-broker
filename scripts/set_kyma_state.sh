#!/bin/bash

KYMA_ID=$1
STATE=$2

if [ -z "$KYMA_ID" ] || [ -z "$STATE" ]; then
  echo "Usage: $0 <kyma-id> <state>"
  echo "Valid states: Processing, Deleting, Ready, Error, Warning, Unmanaged"
  exit 1
fi

case "$STATE" in
  Processing|Deleting|Ready|Error|Warning|Unmanaged)
    ;;
  *)
    echo "Invalid state: $STATE"
    echo "Valid states: Processing, Deleting, Ready, Error, Warning, Unmanaged"
    exit 1
    ;;
esac

echo "Patching Kyma '$KYMA_ID' in namespace kcp-system to state '$STATE'..."

kubectl patch kyma "$KYMA_ID" \
  -n kcp-system \
  --type merge \
  --subresource status \
  -p "{\"status\": {\"state\": \"$STATE\"}}"

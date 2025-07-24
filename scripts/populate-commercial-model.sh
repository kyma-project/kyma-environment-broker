#!/usr/bin/env bash

# This script populates missing commercial model values in the provisioning parameters
# of service instances stored in a database.
# It has the following arguments:
#   - Auth URL
#   - Service URL
# ./populate-commercial-model.sh auth_url service_url

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

AUTH_URL="$1"
SERVICE_URL="$2"
echo "Auth URL: $AUTH_URL"
echo "Service URL: $SERVICE_URL"

CLIENT_ID=$(kubectl get secret cis-creds-accounts -n kcp-system -o jsonpath="{.data.id}" | base64 -d)
CLIENT_SECRET=$(kubectl get secret cis-creds-accounts -n kcp-system -o jsonpath="{.data.secret}" | base64 -d)
DB_NAME=$(kubectl get secret kcp-postgresql -n kcp-system -o jsonpath="{.data.postgresql-broker-db-name}" | base64 -d)
DB_USER=$(kubectl get secret kcp-postgresql -n kcp-system -o jsonpath="{.data.postgresql-broker-username}" | base64 -d)
DB_PASS=$(kubectl get secret kcp-postgresql -n kcp-system -o jsonpath="{.data.postgresql-broker-password}" | base64 -d)

echo "Starting port-forwarding..."
kubectl port-forward -n kcp-system deployment/kcp-kyma-environment-broker 8080:8080 5432:5432 > /dev/null 2>&1 &
PF_PID=$!
cleanup() {
  echo "Stopping port-forwarding..."
  kill "$PF_PID"
}
trap cleanup EXIT
sleep 5

QUERY_RESULT=$(PGPASSWORD=$DB_PASS psql -h localhost -p 5432 -U "$DB_USER" -d "$DB_NAME" -t -A -F',' -c "
SELECT instance_id, global_account_id
FROM instances
WHERE instances.provisioning_parameters::json->'ers_context'->'commercial_model' IS NULL;
")

echo "Number of instances without a commercial model: $(echo "$QUERY_RESULT" | grep -c '^')"

ACCESS_TOKEN=$(curl -s -X POST "$AUTH_URL" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -u "$CLIENT_ID:$CLIENT_SECRET" \
  -d "grant_type=client_credentials" | jq -r '.access_token')

UPDATED_COUNT=0
NULL_ACCOUNTS=()

while IFS=',' read -r INSTANCE_ID GLOBAL_ACCOUNT_ID; do
  if [[ -z "$INSTANCE_ID" || -z "$GLOBAL_ACCOUNT_ID" ]]; then
    echo "Skipping invalid entry. INSTANCE_ID='$INSTANCE_ID', GLOBAL_ACCOUNT_ID='$GLOBAL_ACCOUNT_ID'" >&2
    continue
  fi

  COMMERCIAL_MODEL=$(curl -s --request GET \
    --url "$SERVICE_URL/accounts/v1/globalAccounts/$GLOBAL_ACCOUNT_ID" \
    --header "Authorization: Bearer $ACCESS_TOKEN" | jq -r '.commercialModel')

  if [[ "$COMMERCIAL_MODEL" == "null" || -z "$COMMERCIAL_MODEL" ]]; then
    NULL_ACCOUNTS+=("$GLOBAL_ACCOUNT_ID")
  else
    echo "Updating instance $INSTANCE_ID with commercial model '$COMMERCIAL_MODEL'"

    UPDATE_QUERY="UPDATE instances SET provisioning_parameters = jsonb_set(provisioning_parameters::jsonb, '{ers_context,commercial_model}', to_jsonb('$COMMERCIAL_MODEL'::text)) WHERE instance_id = '$INSTANCE_ID';"

    PGPASSWORD=$DB_PASS psql -h localhost -p 5432 -U "$DB_USER" -d "$DB_NAME" -c "$UPDATE_QUERY" > /dev/null 2>&1
    UPDATED_COUNT=$((UPDATED_COUNT + 1))
  fi
done <<< "$QUERY_RESULT"

echo -e "\nUpdated $UPDATED_COUNT/$(echo "$QUERY_RESULT" | grep -c '^') instances"

if [[ ${#NULL_ACCOUNTS[@]} -gt 0 ]]; then
  echo -e "\nGlobal Account IDs with null commercial model:"
  printf "%s\n" "${NULL_ACCOUNTS[@]}" | sort -u
fi

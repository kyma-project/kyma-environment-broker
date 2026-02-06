#!/usr/bin/env bash
# Populate the performance test database
# Usage: populate_performance_db.sh

set -o nounset
set -o errexit
set -E
set -o pipefail

DB_NAME=$(kubectl get secret kcp-postgresql -n kcp-system -o jsonpath="{.data.postgresql-broker-db-name}" | base64 -d)
DB_USER=$(kubectl get secret kcp-postgresql -n kcp-system -o jsonpath="{.data.postgresql-broker-username}" | base64 -d)
DB_PASS=$(kubectl get secret kcp-postgresql -n kcp-system -o jsonpath="{.data.postgresql-broker-password}" | base64 -d)

kubectl port-forward -n kcp-system deployment/postgres 5432:5432 &
PORT_FORWARD_PID=$!
sleep 5

PGPASSWORD=$DB_PASS psql -h localhost -p 5432 -U $DB_USER -d $DB_NAME -f resources/installation/migrations/populate_performance_tests_database.up.sql

kill $PORT_FORWARD_PID

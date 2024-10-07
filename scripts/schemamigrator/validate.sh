#!/usr/bin/env bash

# This script is responsible for validating if migrations scripts are correct.
# It starts Postgres, executes UP and DOWN migrations.
# This script requires `kcp-schema-migrator` Docker image.

RED='\033[0;31m'
GREEN='\033[0;32m'
INVERTED='\033[7m'
NC='\033[0m' # No Color

set -e

IMG_NAME="keb-schema-migrator"
NETWORK="migration-test-network"
POSTGRES_CONTAINER="migration-test-postgres"
POSTGRES_VERSION="17"

DB_NAME="broker"
DB_USER="usr"
DB_PWD="pwd"
DB_PORT="5432"
DB_SSL_PARAM="disable"
POSTGRES_MULTIPLE_DATABASES="broker"

# Get the directory of the running script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

function cleanup() {
    echo -e "${GREEN}Cleanup Postgres container and network${NC}"
    docker rm --force ${POSTGRES_CONTAINER}
    docker network rm ${NETWORK}
}

trap cleanup EXIT

echo -e "${GREEN}Create network${NC}"
docker network create --driver bridge ${NETWORK}

docker build -t ${IMG_NAME} -f Dockerfile.schemamigrator .

echo -e "${GREEN}Start Postgres in detached mode${NC}"
docker run -d --name ${POSTGRES_CONTAINER} \
            --network=${NETWORK} \
            -e POSTGRES_USER=${DB_USER} \
            -e POSTGRES_PASSWORD=${DB_PWD} \
            -e POSTGRES_MULTIPLE_DATABASES="${POSTGRES_MULTIPLE_DATABASES}" \
            -v "${SCRIPT_DIR}"/multiple-postgresql-databases.sh:/docker-entrypoint-initdb.d/multiple-postgresql-databases.sh \
            postgres:${POSTGRES_VERSION}

function migrationUP() {
    echo -e "${GREEN}Run UP migrations ${NC}"

    docker run --rm --network=${NETWORK} \
            -e DB_USER=${DB_USER} \
            -e DB_PASSWORD=${DB_PWD} \
            -e DB_HOST=${POSTGRES_CONTAINER} \
            -e DB_PORT=${DB_PORT} \
            -e DB_NAME=${DB_NAME} \
            -e DB_SSL=${DB_SSL_PARAM} \
            -e DIRECTION="up" \
            -v "${SCRIPT_DIR}"/../../resources/keb/migrations:/migrate/new-migrations \
        ${IMG_NAME}

    echo -e "${GREEN}Show schema_migrations table after UP migrations${NC}"
    docker exec ${POSTGRES_CONTAINER} psql -U usr ${DB_NAME} -c "select * from schema_migrations"
}

function migrationDOWN() {
    echo -e "${GREEN}Run DOWN migrations ${NC}"

    docker run --rm --network=${NETWORK} \
            -e DB_USER=${DB_USER} \
            -e DB_PASSWORD=${DB_PWD} \
            -e DB_HOST=${POSTGRES_CONTAINER} \
            -e DB_PORT=${DB_PORT} \
            -e DB_NAME=${DB_NAME} \
            -e DB_SSL=${DB_SSL_PARAM} \
            -e DIRECTION="down" \
            -e NON_INTERACTIVE="true" \
            -v "${SCRIPT_DIR}"/../../resources/keb/migrations:/migrate/new-migrations \
        ${IMG_NAME}

    echo -e "${GREEN}Show schema_migrations table after DOWN migrations${NC}"
    docker exec ${POSTGRES_CONTAINER} psql -U usr ${DB_NAME} -c "select * from schema_migrations"
}

function migrationProcess() {
    echo -e "${GREEN}Migrations for broker database${NC}"
    migrationUP
    migrationDOWN
}

migrationProcess

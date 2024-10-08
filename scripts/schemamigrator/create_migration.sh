#!/usr/bin/env bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

NAME=$1

if [ -z "$NAME" ] ; then
    echo "NAME argument not provided. Usage: ./create_migration [NAME]"
    exit 1
fi

DATE="$(date +%Y%m%d%H%M)"
MIGRATIONS_DIR="../../resources/keb/migrations"
TRANSACTION_STR=$'BEGIN;\nCOMMIT;'

echo "$TRANSACTION_STR" > "${MIGRATIONS_DIR}/${DATE}_${NAME}.up.sql"
echo "$TRANSACTION_STR" > "${MIGRATIONS_DIR}/${DATE}_${NAME}.down.sql"

#!/usr/bin/env bash
# Usage: check_env_alphabetical_order.sh <yaml_path> <label> [<end_section_regex>]
# Default end_section_regex is 'volumeMounts:' except for deployment.yaml (KEB), which is 'ports:'

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

YAML_PATH="$1"
LABEL="$2"
END_SECTION_REGEX="${3:-volumeMounts}"

TMP_ENV_VARS="/tmp/env_vars_${LABEL}.txt"
TMP_ENV_VARS_SORTED="/tmp/env_vars_${LABEL}_sorted.txt"

awk "/env:/ {flag=1; next} /$END_SECTION_REGEX:/ {flag=0} flag && /^ *- name:/" "$YAML_PATH" | sed 's/^ *- name: //' > "$TMP_ENV_VARS"
sort -d "$TMP_ENV_VARS" > "$TMP_ENV_VARS_SORTED"
if ! diff -u --color=always "$TMP_ENV_VARS" "$TMP_ENV_VARS_SORTED"; then
  echo "Environment variables in $LABEL are not sorted alphabetically!"
  exit 1
fi
echo "Environment variables in $LABEL are sorted alphabetically."

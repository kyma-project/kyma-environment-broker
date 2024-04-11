#!/usr/bin/env bash

# This script has the following argument:
#     - releaseID (mandatory)
#     - packed KEB Chart path name (mandatory)
# ./upload_assets.sh 12345678 keb-0.0.0.tgz

RELEASE_ID=${1}
KEB_CHART=${2}

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

# Expected variables:
#   BOT_GITHUB_TOKEN              - github token used to upload the asset
#   KYMA_ENVIRONMENT_BROKER_REPO  - Kyma repository

uploadFile() {
  filePath=${1}
  ghAsset=${2}

  response=$(curl -s -o output.txt -w "%{http_code}" \
                  --request POST --data-binary @"$filePath" \
                  -H "Authorization: token $BOT_GITHUB_TOKEN" \
                  -H "Content-Type: text/yaml" \
                   $ghAsset)
  if [[ "$response" != "201" ]]; then
    echo "::error ::Unable to upload the asset ($filePath): "
    echo "::error ::HTTP Status: $response"
    cat output.txt
    exit 1
  else
    echo "$filePath uploaded"
  fi
}


UPLOAD_URL="https://uploads.github.com/repos/${KYMA_ENVIRONMENT_BROKER_REPO}/releases/${RELEASE_ID}/assets"

echo -e "\n--- Updating GitHub release ${RELEASE_ID} with ${KEB_CHART} asset"

[[ ! -e ${KEB_CHART} ]] && echo "::error ::Packaged KEB chart does not exist" && exit 1

uploadFile "${KEB_CHART}" "${UPLOAD_URL}?name=${KEB_CHART}"

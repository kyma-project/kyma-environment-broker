#!/usr/bin/env bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # must be set if you want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

# This script has the following arguments:
#                       -  image tag - mandatory
#
# ./check_artifacts_existence.sh v2.1.0


export IMAGE_TAG=$1

PROTOCOL=docker://

if [ $(skopeo list-tags ${PROTOCOL}europe-docker.pkg.dev/kyma-project/prod/kyma-environment-broker | jq '.Tags|any(. == env.IMAGE_TAG)') == "true" ]
then
  echo "::warning ::Kyma Environment Broker binary image for tag ${IMAGE_TAG} already exists"
else
  echo "No previous Kyma Environment Broker binary image found for tag ${IMAGE_TAG}"
fi

if [ $(skopeo list-tags ${PROTOCOL}europe-docker.pkg.dev/kyma-project/prod/kyma-environment-deprovision-retrigger-job | jq '.Tags|any(. == env.IMAGE_TAG)') == "true" ]
then
  echo "::warning ::Kyma Environment deprovision retrigger job binary image for tag ${IMAGE_TAG} already exists"
else
  echo "No previous Kyma Environment deprovision retrigger job binary image found for tag ${IMAGE_TAG}"
fi

if [ $(skopeo list-tags ${PROTOCOL}europe-docker.pkg.dev/kyma-project/prod/kyma-environment-runtime-reconciler | jq '.Tags|any(. == env.IMAGE_TAG)') == "true" ]
then
  echo "::warning ::Kyma Environment runtime reconciler binary image for tag ${IMAGE_TAG} already exists"
else
  echo "No previous Kyma Environment runtime reconciler binary image found for tag ${IMAGE_TAG}"
fi

if [ $(skopeo list-tags ${PROTOCOL}europe-docker.pkg.dev/kyma-project/prod/kyma-environment-subaccount-cleanup-job | jq '.Tags|any(. == env.IMAGE_TAG)') == "true" ]
then
  echo "::warning ::Kyma Environment subaccount cleanup job binary image for tag ${IMAGE_TAG} already exists"
else
  echo "No previous Kyma Environment subaccount cleanup job binary image found for tag ${IMAGE_TAG}"
fi

if [ $(skopeo list-tags ${PROTOCOL}europe-docker.pkg.dev/kyma-project/prod/kyma-environment-subscription-cleanup-job | jq '.Tags|any(. == env.IMAGE_TAG)') == "true" ]
then
  echo "::warning ::Kyma Environment subscription cleanup job binary image for tag ${IMAGE_TAG} already exists"
else
  echo "No previous Kyma Environment subscription cleanup job binary image found for tag ${IMAGE_TAG}"
fi

if [ $(skopeo list-tags ${PROTOCOL}europe-docker.pkg.dev/kyma-project/prod/kyma-environment-trial-cleanup-job | jq '.Tags|any(. == env.IMAGE_TAG)') == "true" ]
then
  echo "::warning ::Kyma Environment trial cleanup job binary image for tag ${IMAGE_TAG} already exists"
else
  echo "No previous Kyma Environment trial cleanup job binary image found for tag ${IMAGE_TAG}"
fi

if [ $(skopeo list-tags ${PROTOCOL}europe-docker.pkg.dev/kyma-project/prod/kyma-environments-cleanup-job | jq '.Tags|any(. == env.IMAGE_TAG)') == "true" ]
then
  echo "::warning ::Kyma Environments cleanup job binary image for tag ${IMAGE_TAG} already exists"
else
  echo "No previous Kyma Environments cleanup job binary image found for tag ${IMAGE_TAG}"
fi
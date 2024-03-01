#!/usr/bin/env bash

# This script bumps the KEB images in the chart, utils and the KEB chart version.
# It has the following arguments:
#   - release tag (mandatory)
# ./bump_keb_chart.sh 0.0.0

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked


RELEASE_TAG=$1
VALUES_YAML="resources/keb/values.yaml"

KEYS=$(yq e '.global.images | keys | .[]' $VALUES_YAML | grep 'kyma_environment')

# bump images in resources/keb/values.yaml
for key in $KEYS
do
    # yq removes empty lines when editing files, so the diff and patch are used to preserve formatting.
    yq e ".global.images.$key.version = \"$RELEASE_TAG\"" $VALUES_YAML > $VALUES_YAML.new
    yq '.' $VALUES_YAML > $VALUES_YAML.noblanks
    diff -B $VALUES_YAML.noblanks $VALUES_YAML.new > resources/keb/patch.file
    patch $VALUES_YAML resources/keb/patch.file 
    rm $VALUES_YAML.noblanks
    rm resources/keb/patch.file
    rm $VALUES_YAML.new
done

yq e ".spec.jobTemplate.spec.template.spec.containers[0].image = \"europe-docker.pkg.dev/kyma-project/prod/kyma-environments-cleanup-job:$RELEASE_TAG\"" -i utils/kyma-environments-cleanup-job/kyma-environments-cleanup-job.yaml
yq e ".version = \"$RELEASE_TAG\"" -i resources/keb/Chart.yaml
yq e ".appVersion = \"$RELEASE_TAG\"" -i resources/keb/Chart.yaml


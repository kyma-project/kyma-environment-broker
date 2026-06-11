#!/usr/bin/env bash

# Updates component-config.yaml with the given release tag.
# Usage: ./scripts/create_component_config.sh <release-tag>

TAG=${1}

# standard bash error handling
set -o nounset
set -o errexit
set -E
set -o pipefail

echo "Updating component-config.yaml for release ${TAG}:"

cat <<EOF | tee component-config.yaml
name: kyma-project.io/kyma-runtime/kcp-components/keb-sap
team: kyma/gopher
images:
  - europe-docker.pkg.dev/kyma-project/prod/kyma-environment-broker:${TAG}
  - europe-docker.pkg.dev/kyma-project/prod/kyma-environment-deprovision-retrigger-job:${TAG}
  - europe-docker.pkg.dev/kyma-project/prod/kyma-environment-runtime-reconciler:${TAG}
  - europe-docker.pkg.dev/kyma-project/prod/kyma-environment-expirator-job:${TAG}
  - europe-docker.pkg.dev/kyma-project/prod/kyma-environment-subaccount-cleanup-job:${TAG}
  - europe-docker.pkg.dev/kyma-project/prod/kyma-environment-subaccount-sync:${TAG}
  - europe-docker.pkg.dev/kyma-project/prod/kyma-environment-broker-schema-migrator:${TAG}
  - europe-docker.pkg.dev/kyma-project/prod/kyma-environment-service-binding-cleanup-job:${TAG}
EOF

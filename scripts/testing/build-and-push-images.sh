#!/usr/bin/env bash

# Script to build all KEB Docker images locally and push them to k3s registry
# Usage: ./build-and-push-images.sh [VERSION]

VERSION=${1:-"PR-999"}
REGISTRY="localhost:5000"

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

echo "Building and pushing Docker images with version: ${VERSION}"

# Array of images to build: name, dockerfile, build-args
declare -a IMAGES=(
    "kyma-environment-broker:Dockerfile.keb:VERSION=${VERSION}"
    "kyma-environments-cleanup-job:Dockerfile.job:BIN=environmentscleanup"
    "kyma-environment-deprovision-retrigger-job:Dockerfile.job:BIN=deprovisionretrigger"
    "kyma-environment-expirator-job:Dockerfile.job:BIN=expirator"
    "kyma-environment-runtime-reconciler:Dockerfile.runtimereconciler:BIN=runtime-reconciler"
    "kyma-environment-subaccount-cleanup-job:Dockerfile.job:BIN=accountcleanup"
    "kyma-environment-subaccount-sync:Dockerfile.subaccountsync:BIN=subaccount-sync"
    "kyma-environment-globalaccounts:Dockerfile.globalaccounts:BIN=globalaccounts"
    "kyma-environment-broker-schema-migrator:Dockerfile.schemamigrator:"
    "kyma-environment-service-binding-cleanup-job:Dockerfile.job:BIN=servicebindingcleanup"
)

for IMAGE_SPEC in "${IMAGES[@]}"; do
    IFS=':' read -r IMAGE_NAME DOCKERFILE BUILD_ARGS <<< "$IMAGE_SPEC"

    echo "Building ${IMAGE_NAME}..."

    # Construct docker build command
    BUILD_CMD="docker build -t ${REGISTRY}/${IMAGE_NAME}:${VERSION} -f ${DOCKERFILE} ."

    # Add build args if they exist
    if [ -n "$BUILD_ARGS" ]; then
        BUILD_CMD="${BUILD_CMD} --build-arg ${BUILD_ARGS}"
    fi

    # Build the image
    eval $BUILD_CMD

    echo "Pushing ${IMAGE_NAME}:${VERSION} to registry..."
    docker push ${REGISTRY}/${IMAGE_NAME}:${VERSION}

    echo "✓ ${IMAGE_NAME}:${VERSION} built and pushed successfully"
    echo ""
done

echo "All images built and pushed successfully!"
echo "Images are available in k3s registry at ${REGISTRY}"

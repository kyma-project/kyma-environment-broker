#!/bin/bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

PR_NUMBER=$1

REPOSITORY=${REPOSITORY:-kyma-project/kyma-environment-broker}
HEAD_SHA=$(gh pr view $PR_NUMBER --repo $REPOSITORY --json headRefOid --jq '.headRefOid')

if [ -z "$HEAD_SHA" ]; then
  echo "Failed to get the head SHA of the pull request"
  exit 1
fi

sleep 30
while true; do
  WORKFLOW_RUN=$(gh run list --repo $REPOSITORY --json name,status,conclusion,createdAt,headSha --jq '[.[] | select(.name == "pull-build-and-test-images" and .headSha == "'"$HEAD_SHA"'")] | sort_by(.createdAt) | last | {name: .name, status: .status, conclusion: .conclusion, created_at: .createdAt}')
  CONCLUSION=$(echo $WORKFLOW_RUN | jq -r '.conclusion')
  STATUS=$(echo $WORKFLOW_RUN | jq -r '.status')

  if [ "$CONCLUSION" == "in_progress" ]; then
    echo "Image build in progress"
    sleep 30
  elif [ "$CONCLUSION" == "success" ]; then
    echo "Images built successfully"
    break
  else
    echo "Image build failed or not ready"
    exit 1
  fi
done
#!/usr/bin/env bash
# move changes to the dedicated branch created from the remote main

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

# Expected variables passed (passed from CI):
#   GH_TOKEN                      - GitHub token for GitHub CLI
#   GIT_EMAIL                     - email setting for PR to be created
#   GIT_NAME                      - user name setting for PR to be created
#   KYMA_ENVIRONMENT_BROKER_REPO  - Kyma repository
#   BUMP_CHART_AND_IMAGES         - branch with updated sec-scanners-config and KEB Chart

TAG=$1

# add changed files to stage
git add sec-scanners-config.yaml
git add resources/keb/values.yaml
git add resources/keb/Chart.yaml
git add utils/kyma-environments-cleanup-job/kyma-environments-cleanup-job.yaml
git add docs/contributor/02-70-chart-config.md

if [ -d "notable-changes/${TAG}" ]; then
   echo "Adding notable changes to the PR"
   git add notable-changes/${TAG}/*
   find notable-changes/${TAG}/ -type f -exec sh -c 'git add -u "notable-changes-to-release/$(basename {})"' \;
fi

#stash staged changes
git stash push --staged

#pass changes to branch created from main
git checkout --force -B main refs/remotes/origin/main
git checkout -B ${BUMP_CHART_AND_IMAGES}

#apply stashed changes
git stash apply
git add sec-scanners-config.yaml
git add resources/keb/values.yaml
git add resources/keb/Chart.yaml
git add utils/kyma-environments-cleanup-job/kyma-environments-cleanup-job.yaml
git add docs/contributor/02-70-chart-config.md

if [ -d "notable-changes/${TAG}" ]; then
   echo "Adding notable changes to the PR"
   git add notable-changes/${TAG}/*
   find notable-changes/${TAG}/ -type f -exec sh -c 'git add -u "notable-changes-to-release/$(basename {})"' \;
fi

#configure git
git config --global user.email ${GIT_EMAIL}
git config --global user.name ${GIT_NAME}

#commit and push changes
git commit -m "Bump sec-scanners-config.yaml, KEB images and Chart to ${TAG}"
git remote set-url origin https://x-access-token:${GH_TOKEN}@github.com/${KYMA_ENVIRONMENT_BROKER_REPO}.git
git push --set-upstream origin ${BUMP_CHART_AND_IMAGES} -f

#create PR
pr_link=$(gh pr create -B main --title "Bump sec-scanners-config.yaml, KEB images and Chart to ${TAG}" --body "https://github.com/${KYMA_ENVIRONMENT_BROKER_REPO}/releases/tag/${TAG}" | tail -n 1)
echo "Link for created PR: ${pr_link}"

pr_number=$(echo "$pr_link" | awk -F'/' '{print $NF}')
gh pr edit $pr_number --add-label kind/enhancement
echo "$pr_number"

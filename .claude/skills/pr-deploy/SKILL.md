---
name: pr-deploy
description: Creates a pr-deploy PR in kyma/management-plane-charts to deploy a KEB PR to the kcp-dev cluster.
---

# pr-deploy

Create a short-lived deployment PR in `kyma/management-plane-charts` that overrides all KEB image tags to a `PR-<n>` dev build so ArgoCD deploys it to `kcp-dev`.

## Usage

```
/pr-deploy <KEB-PR-number>
```

**Examples:**
- `/pr-deploy 3201`
- `/pr-deploy 3205`

---

## What to do

### 1. Resolve the KEB PR number

If the user provided a number, use it directly. Otherwise, run:

```bash
gh pr view --repo kyma-project/kyma-environment-broker --json number --jq .number
```

to get the current branch's open PR number.

Call this `<PR>` for the rest of the skill.

### 2. Ask for confirmation

> You are about to create a pr-deploy PR in `kyma/management-plane-charts` for KEB PR-<PR>. This will deploy PR-<PR> images to kcp-dev. Continue?

Stop if the user says no.

### 3. Get the current chart version

Use `mcp__github-tools__get_file_contents` to read `keb-sap/Chart.yaml` from `owner=kyma`, `repo=management-plane-charts`, `ref=refs/heads/chart/keb-sap`.

Extract the current `version:` field. Increment the patch version by 1. Call this `<NEW_VERSION>`.

Example: `1.5.66` → `1.5.67`

### 4. Get the current values.yaml SHA

Use `mcp__github-tools__get_file_contents` to read `keb-sap/values.yaml` from `owner=kyma`, `repo=management-plane-charts`, `ref=refs/heads/chart/keb-sap`.

Note the file's SHA — you will need it for the update step.

### 5. Create the pr-deploy branch

Use `mcp__github-tools__create_branch` to create a new branch named `keb-pr-<PR>-deploy` from `chart/keb-sap`:

```
owner: kyma
repo: management-plane-charts
branch: keb-pr-<PR>-deploy
from_branch: chart/keb-sap
```

### 6. Update Chart.yaml

Use `mcp__github-tools__get_file_contents` to read `keb-sap/Chart.yaml` (note its SHA), then use `mcp__github-tools__create_or_update_file` to bump the version:

- Replace the `version:` line with `<NEW_VERSION>`
- Commit message: `Bump keb-sap chart version to <NEW_VERSION>`
- Branch: `keb-pr-<PR>-deploy`

Provide the SHA of the existing `Chart.yaml` file.

### 7. Append image overrides to values.yaml

Append the following block to the **end** of `keb-sap/values.yaml` (after the existing content, preceded by a blank line):

```yaml

  # KEB PR-<PR> dev deploy overrides
  images:
    container_registry:
      path: europe-docker.pkg.dev/kyma-project/dev
    kyma_environment_broker:
      dir:
      version: PR-<PR>
    kyma_environment_broker_schema_migrator:
      dir:
      version: PR-<PR>
    kyma_environments_subaccount_cleanup_job:
      dir:
      version: PR-<PR>
    kyma_environment_expirator_job:
      dir:
      version: PR-<PR>
    kyma_environment_deprovision_retrigger_job:
      dir:
      version: PR-<PR>
    kyma_environment_runtime_reconciler:
      dir:
      version: PR-<PR>
    kyma_environment_subaccount_sync:
      dir:
      version: PR-<PR>
    kyma_environment_service_binding_cleanup_job:
      dir:
      version: PR-<PR>
    kyma_environment_analytics:
      dir:
      version: PR-<PR>
```

Note: the block must be indented under `global:` — it sits at the same indentation level as the existing `global.images` keys. The safest approach: read the full current content, append the block, then write the whole file back using `mcp__github-tools__create_or_update_file`.

Use the SHA from step 4.

Commit message: `Add KEB PR-<PR> image overrides`

Branch: `keb-pr-<PR>-deploy`

### 8. Get the GitHub username

```bash
gh api user --jq .login
```

### 9. Create the PR

Use `mcp__github-tools__create_pull_request`:

```
owner: kyma
repo: management-plane-charts
title: Test KEB PR-<PR>
head: keb-pr-<PR>-deploy
base: chart/keb-sap
draft: true
body: |
  ## Summary

  Deploy KEB [PR-<PR>](https://github.com/kyma-project/kyma-environment-broker/pull/<PR>) images to kcp-dev for testing.

  All KEB component images are redirected to `europe-docker.pkg.dev/kyma-project/dev` at tag `PR-<PR>`.

  **Do not merge.** Close this PR when testing is complete.
```

### 10. Apply the `pr-deploy` label

Use `mcp__github-tools__issue_write` with method `update` to add the `pr-deploy` label to the newly created PR (pass the PR number as `issue_number`).

### 11. Return the result

Report the PR URL to the user.

---

## Rules

- Always target `base: chart/keb-sap` — never `main`.
- The PR is **never merged** — it is closed after testing.
- Never add issue-closing keywords in the body.
- Use MCP GitHub tools (`mcp__github-tools__*`) for all `kyma/management-plane-charts` operations — never `gh` CLI for that repo.
- Use `gh` CLI only for `kyma-project/kyma-environment-broker` operations (get PR number, get GitHub username).
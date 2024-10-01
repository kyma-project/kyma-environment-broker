# GitHub Actions Workflows

## ESLint Workflow

This [workflow](/.github/workflows/run-eslint.yaml) runs the ESLint.

The workflow:
1. Checks out code 
2. Invokes `make lint -C testing/e2e/skr`

## Markdown Link Check Workflow

This [workflow](/.github/workflows/markdown-link-check.yaml) checks for broken links in all Markdown files. It is triggered:
- As a periodic check that runs daily at midnight on the main branch in the repository 
- On every pull request

## Release Workflow

See [Kyma Environment Broker Release Pipeline](04-20-release.md) to learn more about the release workflow.

## Promote KEB to DEV Workflow

This [workflow](/.github/workflows/promote-keb-to-dev.yaml) creates a PR to the `management-plane-charts` repository with the given KEB release version. The default version is the latest KEB release. 

## Label Validator Workflow

This [workflow](/.github/workflows/label-validator.yml) is triggered by PRs on the `main` branch. It checks the labels on the PR and requires that the PR has exactly one of the labels listed in this [file](/.github/release.yml).

## Verify KEB Workflow

This [workflow](/.github/workflows/run-verify.yaml) calls the reusable [workflow](/.github/workflows/run-unit-tests-reusable.yaml) with unit tests.
Besides the tests, it also runs Go-related checks and Go linter.

## Govulncheck Workflow

This [workflow](/.github/workflows/run-govulncheck.yaml) runs the Govulncheck.

## Image Build Workflow

This [workflow](/.github/workflows/pull-build-images.yaml) builds images.

## KEB Chart Install Test

This [workflow](/.github/workflows/run-keb-chart-install-tests.yaml) calls the [reusable workflow](/.github/workflows/run-keb-chart-install-tests-reusable.yaml) to install the KEB chart with the new images in the k3s cluster.

## Auto Merge Workflow

This [workflow](/.github/workflows/auto-merge.yaml) enables the auto-merge functionality on a PR that is not a draft.

## All Cheks Passed Workflow

This [workflow](/.github/workflows/pr-checks.yaml) checks if all jobs, except those excluded in the workflow configuration, have passed. If the workflow is triggered by a PR where the author is the `kyma-gopher-bot`, the workflow ends immediately with success.

## Reusable Workflows

There are reusable workflows created. Anyone with access to a reusable workflow can call it from another workflow.

### Unit Tests

This [workflow](/.github/workflows/run-unit-tests-reusable.yaml) runs the unit tests.
No parameters are passed from the calling workflow (callee).
The end-to-end unit tests use a PostgreSQL database in a Docker container as the default storage solution, which allows 
the execution of SQL statements during these tests. You can switch to in-memory storage 
by setting the **DB_IN_MEMORY_FOR_E2E_TESTS** environment variable to `true`. However, by using PostgreSQL, the tests can effectively perform 
instance details serialization and deserialization, providing a clearer understanding of the impacts and outcomes of these processes.

The workflow:
1. Checks out code and sets up the cache
2. Sets up the Go environment
3. Invokes `make go-mod-check`
4. Invokes `make test`

### KEB Chart  Install Tests

This [workflow](/.github/workflows/run-keb-chart-install-tests-reusable.yaml) installs the KEB chart in the k3s cluster. 
You pass the following parameters from the calling workflow:

| Parameter name  | Required | Description                                                          |
| ------------- | ------------- |----------------------------------------------------------------------|
| **last-k3s-versions**  | no  | number of most recent k3s versions to be used for tests, default = `1` |
| **release**  | no  | determines if the workflow is called from release, default = `true` |
| **version**  | no  | chart version, default = `0.0.0.0` |


The workflow:
1. Checks if the KEB chart is rendered successfully by Helm
2. Fetches the **last-k3s-versions** tag versions of k3s releases 
3. Prepares the **last-k3s-versions** k3s clusters with the Docker registries using the list of versions from the previous step
4. Creates required namespaces
5. Installs required dependencies by the KEB chart
6. Installs the KEB chart in the k3s cluster using `helm install`
7. Waits for all tests to finish

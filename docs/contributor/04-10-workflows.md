# GitHub Actions Workflows

## ESLint Workflow

This [workflow](/.github/workflows/run-eslint.yaml) is triggered by PRs on the `main` branch. It runs the ESLint.

The workflow:
- Checks out code 
- Invokes `make lint -C testing/e2e/skr`

## Markdown Link Check Workflow

This [workflow](/.github/workflows/markdown-link-check.yaml) checks for broken links in all Markdown files. It is triggered:
- As a periodic check that runs daily at midnight on the main branch in the repository 
- On every pull request that creates new Markdown files or introduces changes to the existing ones

## Release Workflow

See [Kyma Environment Broker Release Pipeline](04-20-release.md) to learn more about the release workflow.

## Label Validator Workflow

This [workflow](/.github/workflows/label-validator.yml) is triggered by PRs on the `main` branch. It checks the labels on the PR and requires that the PR has exactly one of the labels listed in this [file](/.github/release.yml).

## Unit Tests Workflow

This [workflow](/.github/workflows/run-unit-tests.yaml) is triggered by PRs on the `main` branch. Then it calls the reusable [workflow](/.github/workflows/run-unit-tests-reusable.yaml).

## KEB Chart Tests Workflow

This [workflow](/.github/workflows/run-keb-chart-tests.yaml) is triggered by PRs on the `main` branch. Then it calls the reusable [workflow](/.github/workflows/run-keb-chart-tests-reusable.yaml). 

## Reusable Workflows

There are reusable workflows created. Anyone with access to a reusable workflow can call it from another workflow.

### KEB Chart Tests

This [workflow](/.github/workflows/run-keb-chart-tests-reusable.yaml) applies the KEB chart on the k3s cluster. 
You pass the following parameters from the calling workflow:

| Parameter name  | Required | Description                                                          |
| ------------- | ------------- |----------------------------------------------------------------------|
| **last-k3s-versions**  | no  | number of most recent k3s versions to be used for tests, default = `1` |


The workflow:
- Checks if the KEB chart is rendered by Helm
- Fetches the **last-k3s-versions** tag versions of k3s releases 
- Prepares the **last-k3s-versions** k3s clusters with the Docker registries using the list of versions from the previous step
- Creates required namespaces
- Installs required dependencies by the KEB charts
- Renders and applies the KEB chart on the k3s cluster
- Waits for all tests to finish

### Unit Tests

This [workflow](/.github/workflows/run-unit-tests-reusable.yaml) runs the unit tests.
No parameters are passed from the calling workflow (callee).

The workflow:
- Checks out code and sets up the cache
- Sets up the Go environment
- Invokes `make go-mod-check`
- Invokes `make test`

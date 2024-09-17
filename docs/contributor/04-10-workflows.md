# GitHub Actions Workflows

## ESLint Workflow

This [workflow](/.github/workflows/run-eslint.yaml) runs the ESLint. It is triggered by pull requests (PRs) on the `main` branch that change at least one of the following:
- `/.github` directory content
- `/testing/e2e/skr` directory content
- `Makefile` file

The workflow:
- Checks out code 
- Invokes `make lint -C testing/e2e/skr`

## Markdown Link Check Workflow

This [workflow](/.github/workflows/markdown-link-check.yaml) checks for broken links in all Markdown files. It is triggered:
- As a periodic check that runs daily at midnight on the main branch in the repository 
- On every pull request that creates new Markdown files or introduces changes to the existing ones

## Release Workflow

See [Kyma Environment Broker Release Pipeline](04-20-release.md) to learn more about the release workflow.

## Promote KEB to DEV Workflow

This [workflow](/.github/workflows/promote-keb-to-dev.yaml) creates a PR to the `management-plane-charts` repository with the given KEB release version. The default version is the latest KEB release. 

## Label Validator Workflow

This [workflow](/.github/workflows/label-validator.yml) is triggered by PRs on the `main` branch. It checks the labels on the PR and requires that the PR has exactly one of the labels listed in this [file](/.github/release.yml).

## Verify KEB Workflow

This [workflow](/.github/workflows/run-verify.yaml) calls the reusable [workflow](/.github/workflows/run-unit-tests-reusable.yaml) with unit tests.
Besides the tests, it also runs Go-related checks and Go linter. It is triggered by PRs on the `main` branch that change at least one of the following:
- `/.github` directory content
- `/cmd` directory content
- `/common` directory content
- `/files` directory content
- `/internal` directory content
- `/scripts` directory content
- `/utils/edp-registrator` directory content
- `.golangci.yml` file
- Any `Dockerfile.*` file
- `go.mod` file
- `go.sum` file
- `Makefile` file
- Any `*.go` file
- Any `*.sh` file

## Govulncheck Workflow

This [workflow](/.github/workflows/run-govulncheck.yaml) runs the Govulncheck. It is triggered by PRs on the `main` branch that change at least one of the following:
- `/.github` directory content
- `/cmd` directory content
- `/common` directory content
- `/files` directory content
- `/internal` directory content
- `/scripts` directory content
- `/utils/edp-registrator` directory content
- `.golangci.yml` file
- Any `Dockerfile.*` file
- `go.mod` file
- `go.sum` file
- `Makefile` file
- Any `*.go` file
- Any `*.sh` file

## Image Build Workflow

This [workflow](/.github/workflows/pull-build-images.yaml) builds images.

## KEB Chart Install Test

This [workflow](/.github/workflows/run-keb-chart-install-tests.yaml) calls the [reusable workflow](/.github/workflows/run-keb-chart-install-tests-reusable.yaml) to install the KEB chart with the new images in the k3s cluster. 

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
- Checks out code and sets up the cache
- Sets up the Go environment
- Invokes `make go-mod-check`
- Invokes `make test`

### KEB Chart  Install Tests

This [workflow](/.github/workflows/run-keb-chart-install-tests-reusable.yaml) installs the KEB chart in the k3s cluster. 
You pass the following parameters from the calling workflow:

| Parameter name  | Required | Description                                                          |
| ------------- | ------------- |----------------------------------------------------------------------|
| **last-k3s-versions**  | no  | number of most recent k3s versions to be used for tests, default = `1` |
| **release**  | no  | determines if the workflow is called from release, default = `true` |
| **version**  | no  | chart version, default = `0.0.0.0` |


The workflow:
- Checks if the KEB chart is rendered successfully by Helm
- Fetches the **last-k3s-versions** tag versions of k3s releases 
- Prepares the **last-k3s-versions** k3s clusters with the Docker registries using the list of versions from the previous step
- Creates required namespaces
- Installs required dependencies by the KEB chart
- Installs the KEB chart in the k3s cluster using `helm install`
- Waits for all tests to finish

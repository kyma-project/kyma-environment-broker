# GitHub Actions Workflows

## Markdown Link Check Workflow

This [workflow](/.github/workflows/markdown-link-check.yaml) checks for broken links in all Markdown files. It is triggered:

* As a periodic check that runs daily at midnight on the main branch in the repository
* On every pull request

## Release Workflow

See [Kyma Environment Broker Release Pipeline](04-20-release.md) to learn more about the release workflow.

## Promote KEB to DEV Workflow

This [workflow](/.github/workflows/promote-keb-to-dev.yaml) creates a PR to the `management-plane-charts` repository with the given KEB release version. The default version is the latest KEB release.

## Create and Promote Release Workflow

This [workflow](/.github/workflows/create-and-promote-release.yaml) creates a new KEB release and then promotes it to the development environment. It first runs the [release workflow](04-20-release.md), and then creates a PR to the `management-plane-charts` repository with the given KEB release version.

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

This [workflow](/.github/workflows/run-keb-chart-integration-tests.yaml) calls the [reusable workflow](/.github/workflows/run-keb-chart-integration-tests-reusable.yaml) to install the KEB chart with the new images in the k3s cluster.

## Auto Merge Workflow

This [workflow](/.github/workflows/auto-merge.yaml) enables the auto-merge functionality on a PR that is not a draft.

## All Cheks Passed Workflow

This [workflow](/.github/workflows/pr-checks.yaml) checks if all jobs, except those excluded in the workflow configuration, have passed. If the workflow is triggered by a PR where the author is the `kyma-gopher-bot`, the workflow ends immediately with success.

## Validate Database Migrations Workflow

This [workflow](/.github/workflows/pull-validate-schema-migrator.yaml) runs a validation of database migrations performed by Schema Migrator.

The workflow:

1. Checks out code
2. Invokes the [validation script](/scripts/schemamigrator/validate.sh).

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

### KEB Chart Integration Tests

This [workflow](/.github/workflows/run-keb-chart-integration-tests-reusable.yaml) installs the KEB chart in the k3s cluster. It also provisions, updates, and deprovisions an instance. You pass the following parameters from the calling workflow:

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
7. Waits for the KEB Pod to be ready
8. Provisions an instance
9. Updates the instance  
10. Deprovisions the instance  
11. Waits for all tests to finish

### Performance Tests

This [workflow](/.github/workflows/run-performance-tests-reusable.yaml) runs performance tests on the k3s cluster. You pass the following parameters from the calling workflow:

| Parameter name                              | Required | Description                                                                        |
|---------------------------------------------|:--------:|------------------------------------------------------------------------------------|
| **last-k3s-versions**                       |    no    | number of most recent k3s versions to be used for tests, default = `1`             |
| **release**                                 |    no    | determines if the workflow is called from release, default = `true`                |
| **version**                                 |    no    | chart version, default = `0.0.0.0`                                                 |
| **instances-number**                        |    no    | number of instances to be provisioned, default = `100`                             |
| **updates-number**                          |    no    | number of updates on a single instance, default = `300`                            |
| **kim-delay-seconds**                       |    no    | time to wait before transitioning the runtime CR to the Ready state, default = `0` |
| **provisioning-max-step-processing-time**   |    no    | max time to process a step in provisioning queue, default = `30s`                  |
| **provisioning-workers-amount**             |    no    | amount of workers in provisioning queue, default = `25`                            |
| **update-max-step-processing-time**         |    no    | max time to process a step in update queue, default = `30s`                        |
| **update-workers-amount**                   |    no    | amount of workers in update queue, default = `25`                                  |
| **deprovisioning-max-step-processing-time** |    no    | max time to process a step in deprovisioning queue, default = `30s`                |
| **deprovisioning-workers-amount**           |    no    | amount of workers in deprovisioning queue, default = `25`                          |

The workflow performs the following actions for all jobs:
- Fetches the **last-k3s-versions** tag versions of k3s releases
- Prepares the **last-k3s-versions** k3s clusters with the Docker registries using the list of versions from the previous step
- Creates required namespaces
- Installs required dependencies by the KEB chart
- Installs the KEB chart in the k3s cluster using `helm install`
- Waits for the KEB Pod to be ready
- Populates database with a thousand of instances
- Starts metrics collector

<details>
<summary>Concurrent Provisioning Test</summary>

- **Purpose**: Evaluate KEB performance when handling multiple concurrent provisioning requests.
- **Steps**:
    - Provisions multiple instances.
    - Sets the state of each created runtime to "Ready" after the specified delay.
    - Fetches metrics from `kyma-environment-broker` to measure success rate and average time taken to complete provisioning requests.
    - Fetches metrics such as goroutines, file descriptors, memory usage, and database connections from the metrics collector and generates visual summaries using Mermaid charts.
- **The test fails in the following conditions**:
    - Success rate drops below the defined threshold.

</details>

<details>
<summary>Concurrent Update Test</summary>

- **Purpose**: Assess KEB ability to process multiple concurrent updating requests.
- **Steps**:
    - Provisions multiple instances.
    - Sets the state of each created runtime to "Ready".
    - Updates created instances.
    - Fetches metrics from `kyma-environment-broker` to measure success rate of update requests.
    - Fetches metrics such as goroutines, file descriptors, memory usage, and database connections from the metrics collector and generates visual summaries using Mermaid charts.
- **The test fails in the following conditions**:
    - Success rate drops below the defined threshold.

</details>

<details>
<summary>Multiple Updates on a Single Instance Test</summary>

- **Purpose**: Test KEB behavior when processing multiple update requests for a single instance.
- **Steps**:
    - Provisions the instance.
    - Sets the state of created runtime to "Ready".
    - Updates the instance.
    - Fetches metrics from `kyma-environment-broker` to measure success rate of update requests.
    - Fetches metrics such as goroutines, file descriptors, memory usage, and database connections from the metrics collector and generates visual summaries using Mermaid charts.
- **The test fails in the following conditions**:
    - Success rate drops below the defined threshold.

</details>

<details>
<summary>Concurrent Deprovisioning Test</summary>

- **Purpose**: Measure KEB performance when handling multiple concurrent deprovisioning requests.
- **Steps**:
    - Provisions multiple instances.
    - Sets the state of each created runtime to "Ready".
    - Deprovisions created instances.
    - Fetches metrics from `kyma-environment-broker` to measure success rate of deprovisioning requests.
    - Fetches metrics such as goroutines, file descriptors, memory usage, and database connections from the metrics collector and generates visual summaries using Mermaid charts.
- **The test fails in the following conditions**:
    - Success rate drops below the defined threshold.

</details>

<details>
<summary>Mixed Operations Test</summary>

- **Purpose**: Analyze KEB performance when processing a mix of concurrent provisioning, updating, and deprovisioning requests.
- **Steps**:
    - Provisions multiple instances.
    - Sets the state of each created runtime to "Ready".
    - Sends a mix of concurrent provisioning, update, and deprovisioning requests.
    - Sets the state of each created runtime to "Ready" after the specified delay..
    - Fetches metrics from `kyma-environment-broker` to measure success rate of deprovisioning requests.
    - Fetches metrics such as goroutines, file descriptors, memory usage, and database connections from the metrics collector and generates visual summaries using Mermaid charts.
- **The test fails in the following conditions**:
    - Success rate drops below the defined threshold.

</details>

<details>
<summary>Runtimes Endpoint Test</summary>

- **Purpose**: Test KEB efficiency in handling multiple GET Runtimes requests with a database containing thousands of instances and operations.
- **Steps**:
    - Populates the database with 1k, 10k, and 100k instances.
    - Sends repeated GET requests to the `/runtimes` endpoint to measure availability and response times.
    - Fetches metrics such as goroutines, file descriptors, memory usage, and database connections from the metrics collector and generates visual summaries using Mermaid charts.
- **The test fails in the following conditions**:
    - Success rate drops below the defined threshold.

</details>
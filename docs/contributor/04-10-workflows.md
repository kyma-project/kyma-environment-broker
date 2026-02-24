# GitHub Actions Workflows

## Markdown Link Check Workflow

The [`markdown-link-check`](/.github/workflows/markdown-link-check.yaml) workflow checks for broken links in all Markdown files. It is triggered in the following cases:

* As a periodic check that runs daily at midnight on the main branch in the repository
* On every pull request

## Release Workflow

See [Kyma Environment Broker Release Pipeline](04-20-release.md) to learn more about the release workflow.

## Promote KEB to DEV Workflow

The [`promote-keb-to-dev`](/.github/workflows/promote-keb-to-dev.yaml) workflow creates a PR to the `management-plane-charts` repository with the given KEB release version. The default version is the latest KEB release.

## Create and Promote Release Workflow

The [create-and-promote-release](/.github/workflows/create-and-promote-release.yaml) workflow creates a new KEB release and then promotes it to the development environment. It first runs the [release workflow](04-20-release.md), and then creates a PR to the `management-plane-charts` repository with the given KEB release version.

## Label Validator Workflow

The [`label-validator`](/.github/workflows/label-validator.yml) workflow is triggered by PRs on the `main` branch. It checks the labels on the PR and requires that the PR has exactly one of the labels listed in the [`release.yml`](/.github/release.yml) file.

## Verify KEB - Go Workflow

The [`run-verify-go`](/.github/workflows/run-verify-go.yaml) workflow calls the reusable [`run-unit-tests-reusable`](/.github/workflows/run-unit-tests-reusable.yaml) workflow with unit tests, executes Go-related checks (such as dependency and formatting checks) and runs the Go linter. 

## Verify KEB - Docs Workflow

The [`run-verify-docs`](/.github/workflows/run-verify-docs.yaml) workflow verifies that the documentation describing environment variables is up to date.
It uses the [`scripts/python/generate_env_docs.py`](../../scripts/python/generate_env_docs.py) script to ensure that environment variable tables in the documentation are up to date with the Helm chart and the `values.yaml` file. The script extracts environment variables from templates, matches them with descriptions and defaults from `values.yaml`, and based on these, updates the relevant Markdown files.

### Add New Files to the Documentation Check

To add a new documentation file to the `run-verify-docs` check, perform the following steps:

1. Add the template and the corresponding Markdown file to the `SINGLE_JOBS`, `MULTI_JOBS_IN_ONE_TEMPLATE`, or `COMBINED_JOBS_IN_ONE_MD` lists in `generate_env_docs.py`.
2. Ensure the Markdown file contains a table for environment variables, or a section header where the table should be inserted.
3. The script automatically updates the documentation table when run.

Run the script with the following command:

```sh
python3 scripts/python/generate_env_docs.py
```

## Govulncheck Workflow

The [`run-govulncheck`](/.github/workflows/run-govulncheck.yaml) workflow runs the Govulncheck.

## Image Build Workflow

The [`pull-build-images`](/.github/workflows/pull-build-images.yaml) workflow builds images.

## KEB Chart Install Test

The [`run-keb-chart-integration-tests`](/.github/workflows/run-keb-chart-integration-tests.yaml) workflow calls the [`run-keb-chart-integration-tests-reusable`](/.github/workflows/run-keb-chart-integration-tests-reusable.yaml) reusable workflow to install the KEB chart with the new images in the k3s cluster.

## Auto Merge Workflow

The [`auto-merge`](/.github/workflows/auto-merge.yaml) workflow enables the auto-merge functionality on a PR that is not a draft.

## All Checks Passed Workflow

The [`pr-checks`](/.github/workflows/pr-checks.yaml) workflow checks if all jobs, except those excluded in the workflow configuration, have passed. If the workflow is triggered by a PR where the author is the `kyma-gopher-bot`, the workflow ends immediately with success.

## Validate Database Migrations Workflow

The [`pull-validate-schema-migrator`](/.github/workflows/pull-validate-schema-migrator.yaml) workflow runs a validation of database migrations performed by Schema Migrator.

The workflow performs the following steps:

1. Checks out the code
2. Invokes the [validation script](/scripts/schemamigrator/validate.sh)

## Upload Release Logs as Assets Workflow

The [`upload-release-logs`](/.github/workflows/upload-release-logs.yaml) workflow uploads the logs from the release workflow as assets to the corresponding GitHub release. It is triggered on every published release event.

The workflow performs the following steps:

1. Checks out the repository
2. Waits for the "Create and promote release" workflow to finish if it is still in progress
3. Downloads logs from all attempts of the "Create and promote release" workflow for the current release
4. Uploads the downloaded logs as assets to the current GitHub release
5. Uploads the downloaded logs to a GCP bucket

## Reusable Workflows

There are reusable workflows created. Anyone with access to a reusable workflow can call it from another workflow.

### Unit Tests

The [`run-unit-tests-reusable`](/.github/workflows/run-unit-tests-reusable.yaml) workflow runs the unit tests.
No parameters are passed from the calling workflow.
The end-to-end unit tests use a PostgreSQL database in a Docker container as the default storage solution, which allows
the execution of SQL statements during these tests. You can switch to in-memory storage 
by setting the **DB_IN_MEMORY_FOR_E2E_TESTS** environment variable to `true`. However, by using PostgreSQL, the tests can effectively perform
instance details serialization and deserialization, providing a clearer understanding of the impacts and outcomes of these processes.

The workflow performs the following steps:

1. Checks out code and sets up the cache
2. Sets up the Go environment
3. Invokes `make go-mod-check`
4. Invokes `make test`

### KEB Chart Integration Tests

The [`run-keb-chart-integration-tests-reusable`](/.github/workflows/run-keb-chart-integration-tests-reusable.yaml) workflow installs the KEB chart in the k3s cluster. It also provisions, updates, and deprovisions an instance. You pass the following parameters from the calling workflow:

| Parameter name        | Required | Description                                             | Defaults  |
|-----------------------|:--------:|---------------------------------------------------------|:---------:|
| **last-k3s-versions** | no       | Number of most recent k3s versions to be used for tests | `1`       |
| **release**           | no       | Determines if the workflow is called from release       | `true`    |
| **version**           | no       | Chart version                                           | `0.0.0.0` |

The workflow performs the following steps:

1. Checks if the KEB chart is rendered successfully by Helm
2. Fetches the **last-k3s-versions** tag versions of k3s releases 
3. Prepares the **last-k3s-versions** k3s clusters with the Docker registries using the list of versions from the previous step
4. Creates required namespaces
5. Installs dependencies required by the KEB chart
6. Installs the KEB chart in the k3s cluster using `helm install`
7. Waits for the KEB Pod to be ready
8. Provisions an instance
9. Updates the instance  
10. Deprovisions the instance  
11. Waits for all tests to finish

### Performance Tests

The [`run-performance-tests-reusable`](/.github/workflows/run-performance-tests-reusable.yaml) workflow runs performance tests on the k3s cluster. You pass the following parameters from the calling workflow:

| Parameter name                              | Required | Description                                                         | Defaults  |
|---------------------------------------------|:--------:|---------------------------------------------------------------------|:---------:|
| **release**                                 |    no    | Determines if the workflow is called from release                   |  `true`   |
| **version**                                 |    no    | Chart version                                                       | `0.0.0.0` |
| **instances-number**                        |    no    | Number of instances to be provisioned                               |   `100`   |
| **updates-number**                          |    no    | Number of updates on a single instance                              |   `300`   |
| **kim-delay-seconds**                       |    no    | Time to wait before transitioning the runtime CR to the Ready state |    `0`    |
| **provisioning-max-step-processing-time**   |    no    | Max time to process a step in provisioning queue                    |   `30s`   |
| **provisioning-workers-amount**             |    no    | Number of workers in provisioning queue                             |   `25`    |
| **update-max-step-processing-time**         |    no    | Max time to process a step in update queue                          |   `30s`   |
| **update-workers-amount**                   |    no    | Number of workers in update queue                                   |   `25`    |
| **deprovisioning-max-step-processing-time** |    no    | Max time to process a step in deprovisioning queue                  |   `30s`   |
| **deprovisioning-workers-amount**           |    no    | Number of workers in deprovisioning queue                           |   `25`    |

The workflow performs the following actions for all jobs:

1. Fetches the **last-k3s-versions** tag versions of k3s releases
2. Prepares the **last-k3s-versions** k3s clusters with the Docker registries using the list of versions from the previous step
3. Creates required namespaces
4. Installs required dependencies by the KEB chart
5. Installs the KEB chart in the k3s cluster using `helm install`
6. Waits for the KEB Pod to be ready
7. Populates database with a thousand instances
8. Starts metrics collector

The performance tests include the following:

<details>
<summary>Concurrent Provisioning Test</summary>

- **Purpose**: Evaluates KEB's performance when handling multiple concurrent provisioning requests.
- **Steps**:
  1. Provisions multiple instances.
  2. Sets the state of each created runtime to `Ready` after the specified delay.
  3. Fetches metrics from `kyma-environment-broker` to measure success rate and average time taken to complete provisioning requests.
  4. Fetches metrics such as goroutines, file descriptors, memory usage, and database connections from the metrics collector and generates visual summaries using Mermaid charts.
- **The test fails in the following conditions**:
  - Success rate falls below 100%.

</details>

<details>
<summary>Concurrent Update Test</summary>

- **Purpose**: Assesses KEB's ability to process multiple concurrent updating requests.
- **Steps**:
  1. Provisions multiple instances.
  2. Sets the state of each created runtime to `Ready`.
  3. Updates created instances.
  4. Fetches metrics from `kyma-environment-broker` to measure success rate of update requests.
  5. Fetches metrics such as goroutines, file descriptors, memory usage, and database connections from the metrics collector and generates visual summaries using Mermaid charts.
- **The test fails in the following conditions**:
  - Success rate falls below 100%.

</details>

<details>
<summary>Multiple Updates on a Single Instance Test</summary>

- **Purpose**: Tests KEB's behavior when processing multiple update requests for a single instance.
- **Steps**:
  1. Provisions the instance.
  2. Sets the state of the created runtime to `Ready`.
  3. Updates the instance.
  4. Fetches metrics from `kyma-environment-broker` to measure success rate of update requests.
  5. Fetches metrics such as goroutines, file descriptors, memory usage, and database connections from the metrics collector and generates visual summaries using Mermaid charts.
- **The test fails in the following conditions**:
  - Success rate falls below 100%.

</details>

<details>
<summary>Concurrent Deprovisioning Test</summary>

- **Purpose**: Measures KEB's performance when handling multiple concurrent deprovisioning requests.
- **Steps**:
  1. Provisions multiple instances.
  2. Sets the state of each created runtime to `Ready`.
  3. Deprovisions created instances.
  4. Fetches metrics from `kyma-environment-broker` to measure success rate and average time taken to complete deprovisioning requests.
  5. Fetches metrics such as goroutines, file descriptors, memory usage, and database connections from the metrics collector and generates visual summaries using Mermaid charts.
- **The test fails in the following conditions**:
  - Success rate falls below 100%.

</details>

<details>
<summary>Mixed Operations Test</summary>

- **Purpose**: Analyzes KEB's performance when processing a mix of concurrent provisioning, update, and deprovisioning requests.
- **Steps**:
  1. Provisions multiple instances.
  2. Sets the state of each created runtime to `Ready`.
  3. Sends a mix of concurrent provisioning, update, and deprovisioning requests.
  4. Sets the state of each created runtime to `Ready` after the specified delay.
  5. Fetches metrics from `kyma-environment-broker` to measure success rate of provisioning, update, and deprovisioning requests, as well as the time taken to complete provisioning and deprovisioning operations.
  6. Fetches metrics such as goroutines, file descriptors, memory usage, and database connections from the metrics collector and generates visual summaries using Mermaid charts.
- **The test fails in the following conditions**:
  - Success rate of any operation type falls below 100%.

</details>

<details>
<summary>Runtimes Endpoint Test</summary>

- **Purpose**: Tests KEB's efficiency in handling multiple GET Runtimes requests with a database containing thousands of instances and operations.
- **Steps**:
  1. Populates the database with 1k, 10k, and 100k instances.
  2. Sends repeated GET requests to the `/runtimes` endpoint to measure availability and response times.
  3. Fetches metrics such as goroutines, file descriptors, memory usage, and database connections from the metrics collector and generates visual summaries using Mermaid charts.
- **The test fails in the following conditions**:
  - Success rate falls below 100%.

</details>
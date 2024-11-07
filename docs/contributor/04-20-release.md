# Kyma Environment Broker Release Pipeline

## Overview

The Kyma Environment Broker release pipeline creates proper artifacts:

* `kyma-environment-broker` Docker image in the [registry](https://console.cloud.google.com/artifacts/docker/kyma-project/europe/prod/kyma-environment-broker)
* `kyma-environment-deprovision-retrigger` Docker image in the [registry](https://console.cloud.google.com/artifacts/docker/kyma-project/europe/prod/kyma-environment-deprovision-retrigger)
* `kyma-environments-cleanup-job` Docker image in the [registry](https://console.cloud.google.com/artifacts/docker/kyma-project/europe/prod/kyma-environments-cleanup-job)
* `kyma-environment-runtime-reconciler` Docker image in the [registry](https://console.cloud.google.com/artifacts/docker/kyma-project/europe/prod/kyma-environment-runtime-reconciler)
* `kyma-environment-subaccount-cleanup-job` Docker image in the [registry](https://console.cloud.google.com/artifacts/docker/kyma-project/europe/prod/kyma-environment-subaccount-cleanup-job)
* `kyma-environment-archiver-job` Docker image in the [registry](https://console.cloud.google.com/artifacts/docker/kyma-project/europe/prod/kyma-environment-archiver-job)
* `kyma-environment-expirator-job` Docker image in the [registry](https://console.cloud.google.com/artifacts/docker/kyma-project/europe/prod/kyma-environment-expirator-job)
* `kyma-environment-subaccount-sync` Docker image in the [registry](https://console.cloud.google.com/artifacts/docker/kyma-project/europe/prod/kyma-environment-subaccount-sync)

## Run the Pipeline

### Create a Release

![Release diagram](../assets/release.drawio.svg)

To create a release, follow these steps:

1. Run GitHub action **Create release**:
   
   i.  Go to the **Actions** tab  
   ii. Click on **Create release** workflow  
   iii. Click  **Run workflow** on the right  
   iv. Provide a version, for example, 1.2.0  
   
2. The GitHub action asynchronously initiates release validation and unit tests. The validation is done by checking if the GitHub tag already exists, if there are any old Docker images for that GitHub tag, and if merged PRs that are part of this release are labeled correctly. Additionally, it stops the release process if a feature has been added, but only the patch version number has been bumped up.
3. The GitHub action initiates the image builders.
4. The Image builders upload the binary images.
5. The GitHub action initiates KEB chart install test.
6. The GitHub action bumps the security scanner config, KEB images and KEB chart version.
7. The GitHub action creates a GitHub tag and draft release with the provided name.
8. The GitHub action commits the new KEB chart metadata to the `gh-pages` branch.
9. The GitHub action publishes the release.

### Replace an Existing Release

To regenerate an existing release, perform the following steps:

1. Delete the GitHub release.
2. Delete the GitHub tag.
3. Run the [**Create release**](#create-a-release) pipeline.

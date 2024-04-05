# Kyma Environment Broker Release Pipeline

## Overview

The Kyma Environment Broker release pipeline creates proper artifacts:
 - `kyma-environment-broker` Docker image in the [registry](https://console.cloud.google.com/artifacts/docker/kyma-project/europe/prod/kyma-environment-broker)
 - `kyma-environment-deprovision-retrigger` Docker image in the [registry](https://console.cloud.google.com/artifacts/docker/kyma-project/europe/prod/kyma-environment-deprovision-retrigger)
 - `kyma-environments-cleanup-job` Docker image in the [registry](https://console.cloud.google.com/artifacts/docker/kyma-project/europe/prod/kyma-environments-cleanup-job )
 - `kyma-environment-runtime-reconciler` Docker image in the [registry](https://console.cloud.google.com/artifacts/docker/kyma-project/europe/prod/kyma-environment-runtime-reconciler)
 - `kyma-environment-trial-cleanup-job` Docker image in the [registry](https://console.cloud.google.com/artifacts/docker/kyma-project/europe/prod/kyma-environment-trial-cleanup-job)
 - `kyma-environment-subaccount-cleanup-job` Docker image in the [registry](https://console.cloud.google.com/artifacts/docker/kyma-project/europe/prod/kyma-environment-subaccount-cleanup-job)

## Run the Pipeline

### Create a Release

![Release diagram](../assets/release.svg)

To create a release, follow these steps:

1. Run GitHub action **Create release**:  
   i.  Go to the **Actions** tab  
   ii. Click on **Create release** workflow   
   iii. Click  **Run workflow** on the right  
   iv. Provide a version, for example, 1.2.0  
   v. Choose whether to bump or not to bump the security scanner config  
   vi. Enter the number of the last k3s versions to be used for KEB chart tests or leave empty (the default value is `3`)
   
2. The GitHub action, defined in the [`create-release`](/.github/workflows/create-release.yaml) file, validates the release by checking if the GitHub tag already exists, if there are any old Docker images for that GitHub tag, and if merged PRs that are part of this release are labeled correctly. Additionally, it stops the release process if a feature has been added, but only the patch version number has been bumped up.
3. The GitHub action asynchronously initiates unit tests, KEB chart tests. It also asynchronously initiates the bump of the security scanner config if you chose this option in step 1.v. In such a case, the GitHub action creates a PR with a new security scanner config that includes the new GitHub tag version.
4. A code owner approves the PR with the security scanner config bump. 
5. The GitHub action creates a GitHub tag and draft release with the provided name.
6. The GitHub action initiates an await for Prow Jobs status.
7. The tag creation triggers ProwJobs defined in [`kyma-environment-broker-build.yaml`](https://github.com/kyma-project/test-infra/blob/main/prow/jobs/kyma-project/kyma-environment-broker/kyma-environment-broker-build.yaml):
- `post-keb-release-build`
- `post-keb-deprovision-retrigger-job-release-build`
- `post-keb-cleanup-job-release-build`
- `post-keb-runtime-reconciler-job-release-build`
- `post-keb-trial-cleanup-job-release-build` 
- `post-keb-subaccount-cleanup-job-release-build`
8. The ProwJobs upload the binary images.
9. The GitHub action initiates the bump of the KEB images and KEB chart version.
10. A code owner approves the PR with the KEB images and KEB chart version bump.
11. The GitHub action commits the new KEB chart metadata to the `gh-pages` branch.
12. The GitHub action publishes the release.


### Replace an Existing Release

To regenerate an existing release, perform the following steps:

1. Delete the GitHub release.
2. Delete the GitHub tag.
3. Run the [**Create release**](#create-a-release) pipeline.

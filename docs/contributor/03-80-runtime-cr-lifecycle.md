# Runtime Custom Resource Lifecycle

## Overview

Kyma Environment Broker (KEB) creates a Runtime Custom Resource (CR) during the provisioning process. Its specification describes the desired runtime configuration. 
Kyma Infrastructure Manager (KIM) reconciles the Runtime CR state.

## Provisioning

During provisioning, KEB creates the Runtime CR with the desired runtime configuration, which includes information about the cluster, machine types, network configuration, and other settings.
Then, KEB waits for KIM to set the state of the Runtime CR to `Ready`. When the state is set to `Ready`, KEB considers the provisioning process successful.
If KIM fails to set the state of the Runtime CR to `Ready` within the timeout period (currently set to 60 minutes), KEB considers the provisioning process failed and initiates the Runtime CR removal.

## Deprovisioning and Suspension

During the deprovisioning process, KEB removes the Runtime CR and waits till the resource is deleted.

## Unsuspension

When the Kyma runtime is unsuspended, KEB creates a new Runtime CR with the same specification as the previous one. The process is identical to the provisioning process, where KEB waits for KIM to set the state of the new Runtime CR to `Ready`.

## Update

When the Kyma runtime is updated, KEB updates the Runtime CR with the new specification. Then, KEB waits for KIM to set the state of the Runtime CR to either `Ready` or `Failed`. If the state is set to `Ready`, KEB considers the update process successful. If the state is set to `Failed`, KEB considers the update process failed.
If the state of the Runtime CR to is neither `Ready` nor `Failed` KEB waits till the timeout period (currently set to 120 minutes) expires and then considers the update process failed.

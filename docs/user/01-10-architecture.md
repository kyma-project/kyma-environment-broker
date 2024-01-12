# Kyma Environment Broker Architecture

The diagram and steps describe the Kyma Environment Broker (KEB) workflow and the roles of specific components in this process:

![KEB diagram](../assets/keb-arch.svg)

1. The user sends a request to create a new cluster with SAP BTP, Kyma runtime.

2. KEB sends the request to create a new cluster to the Runtime Provisioner component.

3. Provisioner creates a new cluster.

4. KEB creates GardenerCluster resource.

5. Infrastructure Manager creates and maintains a Secret containing a kubeconfig.

6. Kyma Environment Broker creates a cluster configuration in Reconciler (except for the preview plan).

7. Reconciler installs Kyma (except for the preview plan). 

8. KEB creates a Kyma resource.

9. Lifecycle Manager manages Kyma modules.

> **NOTE:** In the future, Provisioner and Reconciler will be deprecated.  KEB will then integrate with Infrastructure Manager. To learn about the planned KEB workflow, read [Kyma Environment Broker Target Architecture](01-20-target-architecture.md).

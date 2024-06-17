# Kyma Environment Broker Architecture

The diagram and steps describe the Kyma Environment Broker (KEB) workflow and the roles of specific components in this process:

![KEB diagram](../assets/keb-arch.drawio.svg)

1. The user sends a request to create a new cluster with SAP BTP, Kyma runtime.

2. KEB sends the request to create a new cluster to Runtime Provisioner.

3. Runtime Provisioner creates a new cluster.

4. KEB creates a GardenerCluster resource.

5. Infrastructure Manager creates and maintains a Secret containing a kubeconfig.

6. KEB creates a Kyma resource.

7. Lifecycle Manager manages Kyma modules.

> [!NOTE] 
> In the future, Kyma Runtime Provisioner will be deprecated.  KEB will then integrate with Infrastructure Manager. To learn about the planned KEB workflow, read [Kyma Environment Broker Target Architecture](01-20-target-architecture.md).

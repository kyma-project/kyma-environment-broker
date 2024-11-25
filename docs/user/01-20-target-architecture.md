# Kyma Environment Broker Target Architecture

> [!NOTE]
> After Runtime Provisioner is deprecated, Kyma Environment Broker (KEB) will integrate with Infrastructure Manager. The diagram and description in this document present the KEB target architecture. To read about the current KEB workflow, go to [Kyma Environment Broker Architecture](01-10-architecture.md).

![KEB target architecture](../assets/target-keb-arch.drawio.svg)

1. The user sends a request to create a new cluster with SAP BTP, Kyma runtime.
2. KEB creates a GardenerCluster resource.
3. Infrastructure Manager provisions a new Kubernetes cluster.
4. Infrastructure Manager creates and maintains a Secret containing a kubeconfig.
5. KEB creates a Kyma resource.
6. Lifecycle Manager reads the Secret every time it's needed.
7. Lifecycle Manager manages modules within SAP BTP, Kyma runtime.

> [!NOTE]
> Once the planned changes are implemented, the [Orchestration](../contributor/02-50-orchestration.md) document will be deprecated as irrelevant.

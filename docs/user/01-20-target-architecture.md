# Kyma Environment Broker Target Architecture

> [!NOTE] 
> After Runtime Provisioner and Reconciler are deprecated, Kyma Environment Broker (KEB) will integrate with Infrastructure Manager. The diagram and description in this doc present the KEB target architecture. To read about the current KEB workflow, go to [Kyma Environment Broker Architecture](01-10-architecture.md).

![KEB target architecture](../assets/target-keb-arch.svg)

1. The user sends a request to create a new cluster with SAP BTP, Kyma runtime.
2. KEB creates a GardenerCluster resource.
3. Infrastructure Manager provisions a new Kubernetes cluster.
4. Infrastructure Manager creates and maintains a Secret containing a kubeconfig.
5. KEB creates a Kyma resource.
6. Lifecycle Manager reads the Secret every time it's needed.
7. Lifecycle Manager manages modules within SAP BTP, Kyma runtime.

> [!NOTE] 
> Once the planned changes are implemented, the following documents will be deprecated as irrelevant:
> - [Runtime Components](../contributor/02-10-runtime-components.md)
> - [Set Overrides for Kyma Runtime](../contributor/02-20-runtime-overrides.md)
> - [Configure Kyma Version](../contributor/02-30-kyma-versions.md)
> - [Orchestration](../contributor/02-50-orchestration.md)

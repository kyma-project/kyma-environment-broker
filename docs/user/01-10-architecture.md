# Kyma Environment Broker Architecture

![KEB architecture](../assets/target-keb-arch.drawio.svg)

1. The user sends a request to create a new cluster with SAP BTP, Kyma runtime.
2. KEB creates a Runtime resource.
3. Kyma Infrastructure Manager provisions a new Kubernetes cluster.
4. Kyma Infrastructure Manager creates and maintains a Secret containing a kubeconfig.
5. KEB creates a Kyma resource.
6. Lifecycle Manager reads the Secret every time it's needed.
7. Lifecycle Manager manages modules within SAP BTP, Kyma runtime.

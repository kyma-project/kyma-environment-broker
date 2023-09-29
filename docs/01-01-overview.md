# Kyma Environment Broker (KEB) overview

Kyma Environment Broker (KEB) is a component that allows you to provision [SAP BTP, Kyma runtime](https://kyma-project.io/#/?id=kyma-and-sap-btp-kyma-runtime) on clusters provided by third-party providers. In the process, KEB first uses Provisioner to create a cluster. Then, it uses Reconciler and Lifecycle Manager to install Kyma runtime on the cluster.

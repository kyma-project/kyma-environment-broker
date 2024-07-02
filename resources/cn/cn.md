# Install Kyma Environment Broker in the CN Region

This guide will help you install Kyma Environment Broker (KEB) in the CN region.

## Prerequisites

- All necessary images pushed to the proper Docker registry.
- Istio installed on the cluster.

## Installation

1. Set the proper values in the `sql.yaml`, especially the database password.

2. Prepare a Secret with a kubeconfig to the Gardener project:

   ```shell
   KCFG=`cat <file with kubeconfig>`
   kubectl create secret generic gardener-credentials --from-literal=kubeconfig=$KCFG -n kcp-system
   ```

3. Prepare a Secret with credentials for the Docker registry.

   ```shell
   kubectl create secret docker-registry k8s-ecr-login-renew-docker-secret --docker-server=<registry> --docker-username=<username> --docker-password=<password> --docker-email=<email> -n kcp-system
   ```

4. Apply the following YAML file to install KEB:

   ```shell
   kubectl apply -f sql.yaml
   ```

5. Install the KEB chart:

   ```shell
   helm install keb ../keb --namespace kcp-system -f values.yaml
   ```
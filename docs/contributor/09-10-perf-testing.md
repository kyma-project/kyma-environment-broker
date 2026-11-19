<!--{"metadata":{"publish":false}}-->

# Performance Testing KEB with Large Numbers of Shoots

This document describes how to run a performance test for Kyma Environment Broker (KEB) against a local k3d cluster pre-populated with a large number of Gardener Shoot resources.

## Prerequisites

- A running local KEB installation (see [Install Kyma Environment Broker Locally](01-05-local-installation.md))
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [jq](https://jqlang.github.io/jq/)

## Procedure

### 1. Install the Metrics Server

Install the Kubernetes metrics server so that you can monitor KEB memory and CPU usage during the test:

```bash
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
```

Because the local k3d cluster uses self-signed certificates, patch the metrics server to skip kubelet TLS verification:

```bash
kubectl patch deployment metrics-server -n kube-system \
  --type=json \
  -p='[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]'
```

### 2. Create Shoot Resources

Populate the cluster with realistic Gardener Shoot CRs. The script creates shoots in batches and sets their status to `Succeeded` so KEB treats them as healthy:

```bash
./scripts/create-bulk-shoots.sh 1000
```

The default count is 4000. Pass a number as the first argument to create a different amount.

### 3. Set KEB Memory Limits

Increase the memory request and limit on the KEB deployment to match production-like settings:

```bash
kubectl set resources deployment kcp-kyma-environment-broker \
  -n kcp-system \
  --containers=kyma-environment-broker \
  --requests=memory=512Mi \
  --limits=memory=1Gi
```

### 4. Monitor Resource Usage

In a separate terminal, watch KEB memory and CPU consumption in real time:

```bash
watch -n 1 kubectl top pod -n kcp-system -l app.kubernetes.io/name=kyma-environment-broker
```

### 5. Port-Forward KEB

Expose KEB locally so the performance test script can reach it:

```bash
kubectl port-forward -n kcp-system deploy/kcp-kyma-environment-broker 8080:8080
```

### 6. Run the Performance Test

Run the performance test script. The two arguments are the total number of instances to provision and the number to deprovision concurrently:

```bash
./scripts/perf-test.sh 10 10
```

The script:
1. Sends `N` provision requests to KEB.
2. Simulates KIM and KLM by patching Runtime and Kyma CRs to `Ready`.
3. Waits for all provisioning operations to succeed.
4. Creates a Shoot resource for each provisioned instance.
5. Sends `DEPROV_BATCH` deprovision requests concurrently.
6. Waits for all deprovision operations to complete.

#!/bin/bash

# standard bash error handling
set -o nounset  # treat unset variables as an error and exit immediately.
set -o errexit  # exit immediately when a command fails.
set -E          # needs to be set if we want the ERR trap
set -o pipefail # prevents errors in a pipeline from being masked

VERSION=${1:-''}

# Create namespaces
kubectl create namespace kcp-system || true
kubectl create namespace kyma-system || true
kubectl create namespace istio-system || true
kubectl create namespace garden-kyma-dev || true

# Install Istio
helm repo add istio https://istio-release.storage.googleapis.com/charts
helm repo update
helm install istio-base istio/base -n istio-system --set defaultRevision=default

# Install Prometheus Operator for ServiceMonitor
kubectl create -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/master/bundle.yaml

# Install Postgres
kubectl create -f scripts/testing/yaml/postgres -n kcp-system

# Prepare gardener credentials
KUBE_SERVER_IP=$(ifconfig en0 | awk '$1=="inet" {print $2}' || ifconfig eth0 | awk '$1=="inet" {print $2}')
KCFG=$(kubectl config view --minify --raw \
      | sed "s|https://0\.0\.0\.0|https://${KUBE_SERVER_IP}|" \
      | sed "s|https://127\.0\.0\.1|https://${KUBE_SERVER_IP}|" \
       | yq 'del(.clusters[].cluster."certificate-authority-data") | .clusters[].cluster."insecure-skip-tls-verify" = true')
kubectl create secret generic gardener-credentials --from-literal=kubeconfig="$KCFG" -n kcp-system

# Prepare chart for custom KEB version
if [[ -n "$VERSION" ]]; then
  if [[ "$VERSION" == PR* ]]; then
    scripts/bump_keb_chart.sh "$VERSION" "pr"
  else
    scripts/bump_keb_chart.sh "$VERSION" "release"
  fi
fi

# Create custom resource definitions
kubectl apply -f resources/installation/crd/
# As long as KIM does not support alicloud we need to manually add alicloud provider in CRD (resources/installation/crd/kim-temp.yaml line 1217)
#kubectl apply -f https://raw.githubusercontent.com/kyma-project/infrastructure-manager/main/config/crd/bases/infrastructuremanager.kyma-project.io_runtimes.yaml
kubectl apply -f https://raw.githubusercontent.com/kyma-project/lifecycle-manager/refs/heads/main/config/crd/bases/operator.kyma-project.io_kymas.yaml

# Create predefined secrets
kubectl apply -f resources/installation/secrets/

# Create predefined secret bindings
kubectl apply -f resources/installation/secretbindings/

# Create resource templates
kubectl apply -f resources/installation/templates/

# Deploy KEB helm chart
cd resources/keb
if [[ "$VERSION" == PR* ]]; then
  helm install keb ../keb \
    --namespace kcp-system \
    -f ../../scripts/values.yaml \
    --set global.database.embedded.enabled=false \
    --set testConfig.kebDeployment.useAnnotations=true \
    --set global.images.container_registry.path="europe-docker.pkg.dev/kyma-project/dev" \
    --set global.secrets.mechanism=secrets \
    --debug --wait
else
  helm install keb ../keb \
    --namespace kcp-system \
    -f ../../scripts/values.yaml \
    --set global.database.embedded.enabled=false \
    --set testConfig.kebDeployment.useAnnotations=true \
    --set global.secrets.mechanism=secrets \
    --debug --wait
fi

# Check if KEB pod is in READY state
echo "Waiting for kyma-environment-broker pod to be in READY state..."
kubectl wait --namespace kcp-system --for=condition=Ready pod -l app.kubernetes.io/name=kyma-environment-broker --timeout=60s
EXIT_CODE=$?
if [ $EXIT_CODE -ne 0 ]; then
  echo "The kyma-environment-broker pod did not become READY within the timeout."
  echo "Fetching the logs from the pod..."
  POD_NAME=$(kubectl get pod -l app.kubernetes.io/name=kyma-environment-broker -n kcp-system -o jsonpath='{.items[0].metadata.name}')
  kubectl logs $POD_NAME -n kcp-system
  exit 1
fi

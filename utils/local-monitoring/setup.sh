#!/usr/bin/env bash
#
# Deploy a local monitoring stack (VictoriaMetrics + Plutono/Grafana) on the
# current k3d cluster to scrape KEB metrics.
#
# Prerequisites:
#   - k3d cluster running with KEB deployed in kcp-system namespace
#   - kubectl configured to point to the k3d cluster
#
# Usage:
#   ./setup.sh          # deploy the stack
#   ./setup.sh teardown # remove everything
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NAMESPACE="monitoring"

# ── helpers ──────────────────────────────────────────────────────────────────
info()  { printf "\033[1;34m[INFO]\033[0m  %s\n" "$*"; }
ok()    { printf "\033[1;32m[OK]\033[0m    %s\n" "$*"; }
err()   { printf "\033[1;31m[ERR]\033[0m   %s\n" "$*" >&2; }

wait_for_rollout() {
  local deploy=$1
  info "Waiting for $deploy rollout..."
  kubectl rollout status deployment/"$deploy" -n "$NAMESPACE" --timeout=120s
}

# ── teardown ─────────────────────────────────────────────────────────────────
if [[ "${1:-}" == "teardown" ]]; then
  info "Tearing down monitoring stack..."
  kubectl delete namespace "$NAMESPACE" --ignore-not-found
  ok "Monitoring namespace deleted."
  exit 0
fi

# ── deploy ───────────────────────────────────────────────────────────────────
info "Deploying local monitoring stack into namespace '$NAMESPACE'..."

# 1. Namespace
kubectl apply -f "$SCRIPT_DIR/namespace.yaml"

# 2. VictoriaMetrics
kubectl apply -f "$SCRIPT_DIR/victoriametrics.yaml"
kubectl apply -f "$SCRIPT_DIR/victoriametrics-deployment.yaml"

# 3. Dashboard JSON as ConfigMap (from file, avoids YAML-in-YAML quoting issues)
kubectl create configmap plutono-dashboard-keb-json \
  --from-file=keb-dashboard.json="$SCRIPT_DIR/dashboards/keb-dashboard.json" \
  -n "$NAMESPACE" \
  --dry-run=client -o yaml | kubectl apply -f -

# 4. Plutono / Grafana
kubectl apply -f "$SCRIPT_DIR/plutono-datasource.yaml"
kubectl apply -f "$SCRIPT_DIR/plutono-dashboard-provider.yaml"
kubectl apply -f "$SCRIPT_DIR/plutono-deployment.yaml"

# 5. Wait for pods
wait_for_rollout victoriametrics
wait_for_rollout plutono

ok "Monitoring stack deployed!"
echo ""
info "Access the dashboards:"
echo "  Plutono (Grafana):   kubectl port-forward -n $NAMESPACE svc/plutono 3000:3000"
echo "                       then open http://localhost:3000  (admin/admin)"
echo ""
echo "  VictoriaMetrics UI:  kubectl port-forward -n $NAMESPACE svc/victoriametrics 8428:8428"
echo "                       then open http://localhost:8428/vmui"
echo ""
echo "  KEB metrics (raw):   kubectl port-forward -n kcp-system deployment/kcp-kyma-environment-broker 8080:8080"
echo "                       then open http://localhost:8080/metrics"
echo ""
info "The KEB dashboard is auto-provisioned and set as the home dashboard."

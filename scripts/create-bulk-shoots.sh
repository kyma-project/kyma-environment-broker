#!/usr/bin/env bash
# Creates COUNT realistic Shoot CRs in garden-kyma-dev namespace,
# closely matching the structure of example-shoot.yaml.
# Usage: ./scripts/create-bulk-shoots.sh [COUNT]

set -o nounset
set -o errexit
set -E
set -o pipefail

COUNT=${1:-4000}
NAMESPACE="garden-kyma-dev"
BATCH_SIZE=50

CREDENTIALS_BINDINGS=("aws" "aws-2" "azure" "azure-2" "gcp")

provider_regions() {
  case "$1" in
    aws)   echo "us-east-1 eu-west-1 eu-central-1 ap-northeast-1" ;;
    azure) echo "eastus westeurope northeurope japaneast" ;;
    gcp)   echo "us-central1 europe-west1 europe-west3 asia-northeast1" ;;
  esac
}

provider_machines() {
  case "$1" in
    aws)   echo "m7i.large m7i.xlarge m7i.2xlarge" ;;
    azure) echo "Standard_D4s_v5 Standard_D8s_v5 Standard_D16s_v5" ;;
    gcp)   echo "n2-standard-4 n2-standard-8 n2-standard-16" ;;
  esac
}

provider_volume_type() {
  case "$1" in
    aws)   echo "gp3" ;;
    azure) echo "Premium_LRS" ;;
    gcp)   echo "pd-balanced" ;;
  esac
}

provider_seeds() {
  case "$1" in
    aws)   echo "aws-ha-us3 aws-ha-eu1 aws-ha-ap1" ;;
    azure) echo "az-ha-we1 az-ha-ne1 az-ha-eus1" ;;
    gcp)   echo "gcp-ha-uc1 gcp-ha-ew1 gcp-ha-ew3" ;;
  esac
}

random_element() {
  local arr=($@)
  echo "${arr[$((RANDOM % ${#arr[@]}))]}"
}

random_uuid() {
  local hex
  hex=$(LC_ALL=C tr -dc 'a-f0-9' < /dev/urandom | dd bs=1 count=32 2>/dev/null || true)
  echo "${hex:0:8}-${hex:8:4}-${hex:12:4}-${hex:16:4}-${hex:20:12}"
}

echo "Creating ${COUNT} shoots in namespace ${NAMESPACE} in batches of ${BATCH_SIZE}..."

batch=""
batch_count=0
total=0

for i in $(seq 1 "$COUNT"); do
  NAME=$(printf 's%07x' "$i")

  CB=$(random_element "${CREDENTIALS_BINDINGS[@]}")
  case "$CB" in
    aws*)  PROVIDER="aws"   ;;
    azure*) PROVIDER="azure" ;;
    gcp*)  PROVIDER="gcp"   ;;
  esac

  REGION=$(random_element $(provider_regions "$PROVIDER"))
  MACHINE=$(random_element $(provider_machines "$PROVIDER"))
  VOLUME_TYPE=$(provider_volume_type "$PROVIDER")
  SEED=$(random_element $(provider_seeds "$PROVIDER"))
  ACCOUNT_ID=$(random_uuid)
  SUBACCOUNT_ID=$(random_uuid)
  RUNTIME_ID=$(random_uuid)
  K8S_VERSION="1.$(( 28 + RANDOM % 8 )).$(( RANDOM % 10 ))"

  # AWS-specific zone suffixes
  case "$REGION" in
    us-east-1)    ZONES=("us-east-1b" "us-east-1f" "us-east-1c") ;;
    eu-west-1)    ZONES=("eu-west-1a" "eu-west-1b" "eu-west-1c") ;;
    eu-central-1) ZONES=("eu-central-1a" "eu-central-1b" "eu-central-1c") ;;
    ap-northeast-1) ZONES=("ap-northeast-1a" "ap-northeast-1c" "ap-northeast-1d") ;;
    eastus)       ZONES=("1" "2" "3") ;;
    westeurope)   ZONES=("1" "2" "3") ;;
    northeurope)  ZONES=("1" "2" "3") ;;
    japaneast)    ZONES=("1" "2" "3") ;;
    us-central1)  ZONES=("us-central1-a" "us-central1-b" "us-central1-c") ;;
    europe-west1) ZONES=("europe-west1-b" "europe-west1-c" "europe-west1-d") ;;
    europe-west3) ZONES=("europe-west3-a" "europe-west3-b" "europe-west3-c") ;;
    asia-northeast1) ZONES=("asia-northeast1-a" "asia-northeast1-b" "asia-northeast1-c") ;;
    *) ZONES=("a" "b" "c") ;;
  esac

  batch="${batch}
---
apiVersion: core.gardener.cloud/v1beta1
kind: Shoot
metadata:
  name: ${NAME}
  namespace: ${NAMESPACE}
  labels:
    account: ${ACCOUNT_ID}
    subaccount: ${SUBACCOUNT_ID}
    shoot.gardener.cloud/status: healthy
    provider.extensions.gardener.cloud/${PROVIDER}: 'true'
    extensions.extensions.gardener.cloud/shoot-oidc-service: 'true'
    extensions.extensions.gardener.cloud/shoot-cert-service: 'true'
    extensions.extensions.gardener.cloud/shoot-dns-service: 'true'
    extensions.extensions.gardener.cloud/shoot-networking-filter: 'true'
    extensions.extensions.gardener.cloud/shoot-auditlog-service: 'true'
    extensions.extensions.gardener.cloud/shoot-lakom-service: 'true'
    name.seed.gardener.cloud/${SEED}: 'true'
  annotations:
    infrastructuremanager.kyma-project.io/runtime-id: ${RUNTIME_ID}
    infrastructuremanager.kyma-project.io/runtime-generation: '1'
    gardener.cloud/created-by: system:serviceaccount:garden-kyma-dev:compass-provisioner
    shoot.gardener.cloud/tasks: deployInfrastructure
spec:
  dns:
    domain: ${NAME}.dev.kyma.ondemand.com
    providers:
      - domains:
          include:
            - ${NAME}.dev.kyma.ondemand.com
        primary: true
        type: aws-route53
        credentialsRef:
          kind: Secret
          name: aws-route53-secret-dev
          apiVersion: v1
  extensions:
    - type: shoot-networking-filter
      disabled: false
    - type: shoot-cert-service
      providerConfig:
        apiVersion: service.cert.extensions.gardener.cloud/v1alpha1
        kind: CertConfig
        shootIssuers:
          enabled: true
    - type: shoot-dns-service
      providerConfig:
        apiVersion: service.dns.extensions.gardener.cloud/v1alpha1
        kind: DNSConfig
        dnsProviderReplication:
          enabled: true
        syncProvidersFromShootSpecDNS: true
    - type: shoot-oidc-service
      disabled: false
    - type: shoot-auditlog-service
      providerConfig:
        apiVersion: service.auditlog.extensions.gardener.cloud/v1alpha1
        kind: AuditlogConfig
        secretReferenceName: auditlog-credentials
        serviceURL: https://api.auditlog.cf.sap.hana.ondemand.com:8081
        tenantID: ${SUBACCOUNT_ID}
        type: standard
  kubernetes:
    version: ${K8S_VERSION}
    kubeAPIServer:
      auditConfig:
        auditPolicy:
          configMapRef:
            name: audit-policy
      serviceAccountConfig:
        extendTokenExpiration: false
        maxTokenExpiration: 720h0m0s
      requests:
        maxNonMutatingInflight: 400
        maxMutatingInflight: 200
      eventTTL: 1h0m0s
      logging:
        verbosity: 2
      defaultNotReadyTolerationSeconds: 300
      defaultUnreachableTolerationSeconds: 300
    kubeControllerManager:
      nodeCIDRMaskSize: 24
      nodeMonitorGracePeriod: 40s
    kubeScheduler:
      profile: balanced
    kubeProxy:
      mode: IPTables
      enabled: true
    kubelet:
      failSwapOn: true
      kubeReserved:
        cpu: 80m
        memory: 1Gi
        pid: 20k
      imageGCHighThresholdPercent: 50
      imageGCLowThresholdPercent: 40
      serializeImagePulls: true
    verticalPodAutoscaler:
      enabled: true
      evictAfterOOMThreshold: 10m0s
      evictionRateBurst: 1
      evictionRateLimit: -1
      evictionTolerance: 0.5
      recommendationMarginFraction: 0.15
      updaterInterval: 1m0s
      recommenderInterval: 1m0s
      targetCPUPercentile: 0.9
      targetMemoryPercentile: 0.9
  networking:
    type: calico
    providerConfig:
      overlay:
        enabled: false
    pods: 10.96.0.0/13
    nodes: 10.250.0.0/16
    services: 10.104.0.0/13
    ipFamilies:
      - IPv4
  maintenance:
    autoUpdate:
      kubernetesVersion: true
      machineImageVersion: true
    timeWindow:
      begin: 070000+0000
      end: 080000+0000
    autoRotation:
      credentials:
        etcdEncryptionKey:
          rotationPeriod: 672h0m0s
  provider:
    type: ${PROVIDER}
    workers:
      - name: cpu-worker-0
        cri:
          name: containerd
        machine:
          type: ${MACHINE}
          image:
            name: gardenlinux
            version: 2150.10.0
          architecture: amd64
        maximum: 20
        minimum: 3
        maxSurge: 3
        maxUnavailable: 0
        volume:
          type: ${VOLUME_TYPE}
          size: 80Gi
        zones:
          - ${ZONES[0]}
          - ${ZONES[1]}
          - ${ZONES[2]}
        systemComponents:
          allow: true
        machineControllerManager:
          machineDrainTimeout: 15m0s
          maxEvictRetries: 2
        updateStrategy: AutoRollingUpdate
    workersSettings:
      sshAccess:
        enabled: false
  purpose: development
  region: ${REGION}
  seedName: ${SEED}
  resources:
    - name: auditlog-credentials
      resourceRef:
        kind: Secret
        name: auditlog-dev-eu
        apiVersion: v1
  systemComponents:
    coreDNS:
      autoscaling:
        mode: horizontal
    nodeLocalDNS:
      enabled: true
      forceTCPToUpstreamDNS: false
  schedulerName: default-scheduler
  cloudProfile:
    kind: CloudProfile
    name: ${PROVIDER}
  credentialsBindingName: ${CB}
"

  batch_count=$((batch_count + 1))
  total=$((total + 1))

  if [ "$batch_count" -ge "$BATCH_SIZE" ] || [ "$total" -ge "$COUNT" ]; then
    echo "$batch" | kubectl apply -f - > /dev/null

    # Patch status subresource for each shoot in this batch
    for j in $(seq $((total - batch_count + 1)) "$total"); do
      SNAME=$(printf 's%07x' "$j")
      kubectl patch shoot "${SNAME}" -n "${NAMESPACE}" \
        --type merge --subresource status \
        -p "{
          \"status\": {
            \"hibernated\": false,
            \"observedGeneration\": 1,
            \"technicalID\": \"shoot--kyma-dev--${SNAME}\",
            \"seedName\": \"aws-ha-us3\",
            \"credentials\": {\"encryptionAtRest\": {\"provider\": {\"type\": \"aescbc\"}}},
            \"lastOperation\": {
              \"description\": \"Shoot cluster has been successfully reconciled.\",
              \"lastUpdateTime\": \"2026-10-14T22:34:48Z\",
              \"progress\": 100,
              \"state\": \"Succeeded\",
              \"type\": \"Reconcile\"
            },
            \"lastMaintenance\": {
              \"description\": \"All maintenance operations successful.\",
              \"triggeredTime\": \"2026-10-15T07:39:11Z\",
              \"state\": \"Succeeded\"
            },
            \"conditions\": [
              {\"type\": \"APIServerAvailable\", \"status\": \"True\", \"reason\": \"HealthzRequestSucceeded\", \"message\": \"API server /healthz endpoint responded with success status code.\", \"lastTransitionTime\": \"2026-10-14T22:40:12Z\", \"lastUpdateTime\": \"2026-10-14T22:40:12Z\"},
              {\"type\": \"ControlPlaneHealthy\", \"status\": \"True\", \"reason\": \"ControlPlaneRunning\", \"message\": \"All control plane components are healthy.\", \"lastTransitionTime\": \"2026-10-14T22:40:12Z\", \"lastUpdateTime\": \"2026-10-14T22:40:12Z\"},
              {\"type\": \"ObservabilityComponentsHealthy\", \"status\": \"True\", \"reason\": \"ObservabilityComponentsRunning\", \"message\": \"All observability components are healthy.\", \"lastTransitionTime\": \"2026-10-14T22:40:12Z\", \"lastUpdateTime\": \"2026-10-14T22:40:12Z\"},
              {\"type\": \"EveryNodeReady\", \"status\": \"True\", \"reason\": \"EveryNodeReady\", \"message\": \"Every node registered to the cluster is ready.\", \"lastTransitionTime\": \"2026-10-14T22:40:12Z\", \"lastUpdateTime\": \"2026-10-14T22:40:12Z\"},
              {\"type\": \"SystemComponentsHealthy\", \"status\": \"True\", \"reason\": \"SystemComponentsRunning\", \"message\": \"All system components are healthy.\", \"lastTransitionTime\": \"2026-10-14T22:40:12Z\", \"lastUpdateTime\": \"2026-10-14T22:40:12Z\"}
            ],
            \"constraints\": [
              {\"type\": \"UsesUnifiedHTTPProxyPort\", \"status\": \"True\", \"reason\": \"ShootUsesUnifiedHTTPProxyPort\", \"message\": \"Shoot uses http-proxy port 8443 for VPN\", \"lastTransitionTime\": \"2026-10-14T22:40:12Z\", \"lastUpdateTime\": \"2026-10-14T22:40:12Z\"},
              {\"type\": \"HibernationPossible\", \"status\": \"True\", \"reason\": \"HibernationPossible\", \"message\": \"All checks for hibernation passed.\", \"lastTransitionTime\": \"2026-10-14T22:40:12Z\", \"lastUpdateTime\": \"2026-10-14T22:40:12Z\"},
              {\"type\": \"MaintenancePreconditionsSatisfied\", \"status\": \"True\", \"reason\": \"MaintenancePreconditionsSatisfied\", \"message\": \"All preconditions for maintenance are satisfied.\", \"lastTransitionTime\": \"2026-10-14T22:40:12Z\", \"lastUpdateTime\": \"2026-10-14T22:40:12Z\"}
            ],
            \"advertisedAddresses\": [
              {\"name\": \"external\", \"url\": \"https://api.${SNAME}.dev.kyma.ondemand.com\"},
              {\"name\": \"internal\", \"url\": \"https://api.${SNAME}.kyma-dev.internal.canary.k8s.ondemand.com\"},
              {\"name\": \"service-account-issuer\", \"url\": \"https://api.${SNAME}.kyma-dev.internal.canary.k8s.ondemand.com\"}
            ],
            \"networking\": {
              \"pods\": [\"10.96.0.0/13\"],
              \"nodes\": [\"10.250.0.0/16\"],
              \"services\": [\"10.104.0.0/13\"],
              \"egressCIDRs\": [\"34.227.201.185/32\", \"98.80.111.38/32\"]
            }
          }
        }" &>/dev/null || true
    done

    echo "Applied ${total}/${COUNT} shoots..."
    batch=""
    batch_count=0
  fi
done

echo "Done. Created ${total} shoots in namespace ${NAMESPACE}."

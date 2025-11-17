# Kubeconfig Endpoint

## Overview

The Kyma Environment Broker (KEB) provides an endpoint that lets you retrieve the kubeconfig file for accessing an SAP BTP, Kyma runtime.

## HTTP Request

```
GET /kubeconfig/{instance_id}
```
No request body is required.

## Response Structure

The endpoint returns a standard Kubernetes kubeconfig file.
- [**CLUSTER_NAME**](04-05-cluster-name.md) is used as the cluster name, context name, and—when OIDC authentication is configured the user name.
- If no OIDC configuration is present, the users section is omitted entirely.
- When multiple OIDC configurations are provided, user entries are generated using an incremental naming convention:
  - The first user is named **CLUSTER_NAME**
  - The second user is named **CLUSTER_NAME-2**
  - The third user is named **CLUSTER_NAME-3**
  - The pattern continues accordingly for subsequent users.

### Response Body

```yaml
---
apiVersion: v1
kind: Config
current-context: CLUSTER_NAME
clusters:
  - name: CLUSTER_NAME
    cluster:
      certificate-authority-data: CERTIFICATE_DATA
      server: SERVER_URL
contexts:
  - name: CLUSTER_NAME
    context:
      cluster: CLUSTER_NAME
      user: CLUSTER_NAME
users:
  - name: CLUSTER_NAME
    user:
      exec:
        apiVersion: client.authentication.k8s.io/v1beta1
        args:
          - get-token
          - "--oidc-issuer-url=ISSUER_URL"
          - "--oidc-client-id=CLIENT_ID"
          - "--oidc-extra-scope=email"
          - "--oidc-extra-scope=openid"
        command: kubectl-oidc_login
        installHint: |
          kubelogin plugin is required to proceed with authentication
          # Homebrew (macOS and Linux)
          brew install int128/kubelogin/kubelogin

          # Krew (macOS, Linux, Windows and ARM)
          kubectl krew install oidc-login

          # Chocolatey (Windows)
          choco install kubelogin
```

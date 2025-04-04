package kubeconfig

type kubeconfig struct {
	APIVersion     string `yaml:"apiVersion"`
	Kind           string `yaml:"kind"`
	CurrentContext string `yaml:"current-context"`
	Clusters       []struct {
		Name    string `yaml:"name"`
		Cluster struct {
			CertificateAuthorityData string `yaml:"certificate-authority-data"`
			Server                   string `yaml:"server"`
		} `yaml:"cluster"`
	} `yaml:"clusters"`
	Contexts []struct {
		Name    string `yaml:"name"`
		Context struct {
			Cluster string `yaml:"cluster"`
			User    string `yaml:"user"`
		} `yaml:"context"`
	} `yaml:"contexts"`
	Users []struct {
		Name string                 `yaml:"name"`
		User map[string]interface{} `yaml:"user"`
	} `yaml:"users"`
}

const kubeconfigTemplate = `
{{- if not .OIDCConfigs }}
---
apiVersion: v1
kind: Config
current-context: {{ .ContextName }}
clusters:
- name: {{ .ContextName }}
  cluster:
    certificate-authority-data: {{ .CAData }}
    server: {{ .ServerURL }}
contexts:
- name: {{ .ContextName }}
  context:
    cluster: {{ .ContextName }}
{{- else }}
---
apiVersion: v1
kind: Config
current-context: {{ .ContextName }}
clusters:
- name: {{ .ContextName }}
  cluster:
    certificate-authority-data: {{ .CAData }}
    server: {{ .ServerURL }}
contexts:
{{- range .OIDCConfigs }}
- name: {{ .Name }}
  context:
    cluster: {{ $.ContextName }}
    user: {{ .Name }}
{{- end }}
users:
{{- range .OIDCConfigs }}
- name: {{ .Name }}
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      args:
      - get-token
      - "--oidc-issuer-url={{ .IssuerURL }}"
      - "--oidc-client-id={{ .ClientID }}"
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
{{- end }}
{{- end }}
`

const kubeconfigTemplateForKymaBindings = `
---
apiVersion: v1
kind: Config
current-context: {{ .ContextName }}
clusters:
- name: {{ .ContextName }}
  cluster:
    certificate-authority-data: {{ .CAData }}
    server: {{ .ServerURL }}
contexts:
- name: {{ .ContextName }}
  context:
    cluster: {{ .ContextName }}
    user: {{ .ContextName }}
users:
- name: {{ .ContextName }}
  user:
    token: {{ .Token }}
  `

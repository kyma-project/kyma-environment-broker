# Kyma Custom Resource Template Configuration

Kyma Environment Broker (KEB) provisioning process creates Kyma custom resource based on the configured template. KEB needs a ConfigMap with the configuration.

## ConfigMap  

The appropriate ConfigMap is selected by filtering the resources using labels. KEB recognizes the ConfigMaps with configurations when they contain the label:

```yaml
keb-config: "true"
```

> ### Note:
> Each ConfigMap that defines the configuration must have this label assigned.

The actual configuration is stored in ConfigMap's `data` object. Add the `default` key under the `data` object:

```yaml
data:
  default: |-
    kyma-template: |-
      apiVersion: operator.kyma-project.io/v1beta2
      kind: Kyma
      metadata:
        labels:
          "operator.kyma-project.io/managed-by": "lifecycle-manager"
          "operator.kyma-project.io/beta": "true"
        name: tbd
        namespace: kcp-system
      spec:
        channel: regular
        modules:
        - name: module1
        - name: module2
```

You must define the default configuration that is selected when the supported plan key is missing. This means that, for example, if there are no other plan keys under the `data` object, the default configuration applies to all the plans. You do not have to change `tbd` value of the `kyma-template.metadata.name` field because KEB generates the name for Kyma CR during the provisioning operation.

> ### Note:
> The `kyma-template` configuration is required.

See an example of a ConfigMap with the default configuration for Kyma and specific configurations for `plan1`, `plan2`, and `trial`:

```yaml
# keb-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: keb-config
  labels:
    keb-config: "true"
data:
  default: |-
    kyma-template: |-
      apiVersion: operator.kyma-project.io/v1beta2
      kind: Kyma
      metadata:
        labels:
          "operator.kyma-project.io/managed-by": "lifecycle-manager"
          "operator.kyma-project.io/beta": "true"
        name: tbd
        namespace: kcp-system
      spec:
        channel: regular
        modules:
        - name: api-gateway
        - name: istio
  plan1: |-
    kyma-template: |-
      apiVersion: operator.kyma-project.io/v1beta2
      kind: Kyma
      metadata:
        labels:
          "operator.kyma-project.io/managed-by": "lifecycle-manager"
          "operator.kyma-project.io/beta": "true"
        name: tbd
        namespace: kcp-system
      spec:
        channel: fast
        modules:
        - name: api-gateway
        - name: istio
        - name: btp-operator
  plan2: |-
    kyma-template: |-
      apiVersion: operator.kyma-project.io/v1beta2
      kind: Kyma
      metadata:
        labels:
          "operator.kyma-project.io/managed-by": "lifecycle-manager"
          "operator.kyma-project.io/beta": "true"
        name: tbd
        namespace: kcp-system
      spec:
        channel: fast
        modules:
        - name: api-gateway
        - name: istio
        - name: btp-operator
        - name: keda
        - name: serverless
  trial: |-
    kyma-template: |-
      apiVersion: operator.kyma-project.io/v1beta2
      kind: Kyma
      metadata:
        labels:
          "operator.kyma-project.io/managed-by": "lifecycle-manager"
          "operator.kyma-project.io/beta": "true"
        name: tbd
        namespace: kcp-system
      spec:
        channel: regular
        modules: []
```

The content of the ConfigMap is stored in values.yaml as `runtimeConfiguration`. More details about Kyma custom resource you can find in [Kyma CR documentation](https://github.com/kyma-project/lifecycle-manager/blob/main/docs/contributor/resources/01-kyma.md)

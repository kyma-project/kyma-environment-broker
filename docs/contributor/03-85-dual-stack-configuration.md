# Dual-Stack Networking Configuration

The dual-stack networking configuration allows you to provision Kyma instances with both IPv4 and IPv6 protocols simultaneously. You must explicitly enable this feature at the provider level to make the option available during provisioning.

## Provider Configuration

To enable dual-stack networking option for a provider, add the **dualStack** property to the provider configuration in the `providersConfiguration` section of the `values.yaml` file.

```yaml
providersConfiguration:
  aws:
    # enables dual-stack networking option (IPv4 and IPv6)
    dualStack: true
    
  gcp:
    # enables dual-stack networking option (IPv4 and IPv6)
    dualStack: true
```

When **dualStack** is set to `true`, you can include the **dualStack** parameter in your provisioning requests. By default, dual-stack networking is not enabled for new instances unless you explicitly request it.

> [!NOTE]
> Dual-stack networking is supported only for Amazon Web Services and Google Cloud providers.
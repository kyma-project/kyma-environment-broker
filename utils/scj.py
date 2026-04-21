from kubernetes import client, config


NAMESPACE = "garden-kyma-dev"
RESOURCE = "credentialsbindings"
GROUP = "security.gardener.cloud"
VERSION = "v1alpha1"


def run_subscription_cleanup():
    config.load_kube_config()
    api = client.CustomObjectsApi()

    bindings = api.list_namespaced_custom_object(
        group=GROUP,
        version=VERSION,
        namespace=NAMESPACE,
        plural=RESOURCE,
        label_selector="dirty=true",
    )

    for binding in bindings["items"]:
        name = binding["metadata"]["name"]
        print(f"Removing 'dirty' label from {name}...")
        api.patch_namespaced_custom_object(
            group=GROUP,
            version=VERSION,
            namespace=NAMESPACE,
            plural=RESOURCE,
            name=name,
            body={"metadata": {"labels": {"dirty": None, "tenantName": None}}},
        )


if __name__ == "__main__":
    run_subscription_cleanup()

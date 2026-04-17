"""
This file need a requests library, install it with 'pip install requests --include-deps'

This file contains functions which can be used to provision/update/deprovision an instance. It is an initial (alpha) version.
"""

import requests
import subprocess
import uuid

KEB_BASE_URL = "http://localhost:8080"
KEB_SERVICE_ID = "47c9dcbf-ff30-448e-ab36-d3bad66ba281"

_DEFAULT_REGIONS = {
    "aws": "eu-central-1",
    "build-runtime-aws": "eu-central-1",
    "azure": "centralus",
    "azure_lite": "centralus",
    "build-runtime-azure": "centralus",
    "gcp": "europe-west3",
    "build-runtime-gcp": "europe-west3",
    "preview": "eu-central-1",
}

_catalog_cache = None


def _load_catalog():
    global _catalog_cache
    if _catalog_cache is not None:
        return _catalog_cache
    url = f"{KEB_BASE_URL}/oauth/v2/catalog"
    headers = {"X-Broker-API-Version": "2.14"}
    if VERBOSE:
        print(f"Fetching catalog from {url}...")
    response = requests.get(url, headers=headers)
    response.raise_for_status()
    _catalog_cache = {}
    for service in response.json().get("services", []):
        for plan in service.get("plans", []):
            schemas = plan.get("schemas", {}).get("service_instance", {})
            _catalog_cache[plan["name"]] = {
                "id": plan["id"],
                "create_schema": schemas.get("create", {}).get("parameters", {}),
                "update_schema": schemas.get("update", {}).get("parameters", {}),
            }
    if VERBOSE:
        print(f"Catalog loaded. Available plans: {', '.join(sorted(_catalog_cache.keys()))}")
    return _catalog_cache


def _get_plan(plan_name):
    catalog = _load_catalog()
    name = plan_name.lower()
    if name not in catalog:
        available = ", ".join(sorted(catalog.keys()))
        raise ValueError(f"Unknown plan '{plan_name}'. Available: {available}")
    return catalog[name]


def _default_region(plan_name):
    return _DEFAULT_REGIONS.get(plan_name.lower(), "")


def _validate_parameters(schema, parameters):
    errors = []
    for field in schema.get("required", []):
        if field not in parameters:
            errors.append(f"Missing required field: '{field}'")
    props = schema.get("properties", {})
    for key, value in parameters.items():
        if key in props:
            allowed = props[key].get("enum")
            if allowed and value not in allowed:
                errors.append(f"Invalid value '{value}' for '{key}'. Allowed: {allowed}")
    if errors:
        raise ValueError("Parameter validation failed:\n" + "\n".join(f"  - {e}" for e in errors))


class Runtime:
    """
    A class representing a Runtime instance provisioned through the Service Broker API. It groups operations related to a specific instance:
    - update_runtime_status: Update the status of the Runtime CR in Kubernetes (simulate the KIM work)
    - deprovision: Deprovision the instance through the Service Broker API of KEB
    - update: Update the instance through the Service Broker API of KEB

    for example:

    import keb
    runtime = keb.provision()
    runtime.update_runtime_status("Ready")
    runtime.update({"autoScalerMax": 16})
    runtime.deprovision()

    """

    def __init__(self, instance_id, provisioning_operation_id, plan_id, plan_name):
        self.instance_id = instance_id
        self.provisioning_operation_id = provisioning_operation_id
        self.plan_id = plan_id
        self.plan_name = plan_name

    def __str__(self):
        return f"Runtime(instance_id={self.instance_id}, provisioning_operation_id={self.provisioning_operation_id})"

    def update_runtime_status(self, state):
        update_runtime_status(self.instance_id, state)

    def deprovision(self):
        deprovision(self.instance_id, self.plan_id)

    def update(self, parameters={}, validate=False):
        update(self.instance_id, self.plan_id, self.plan_name, parameters, validate=validate)

    def get_instance(self):
        url = f"{KEB_BASE_URL}/oauth/v2/service_instances/{self.instance_id}"
        headers = {
            "X-Broker-API-Version": "2.14",
            "Content-Type": "application/json"
        }
        response = requests.get(url, headers=headers)
        if response.status_code == 200:
            return response.json()
        else:
            print("Failed to get the instance.")
            print("Status Code:", response.status_code)
            print("Response:", response.text)
            return None


def update(instance_id, plan_id, plan_name, parameters={}, validate=False):
    plan = _get_plan(plan_name)
    if validate:
        _validate_parameters(plan["update_schema"], parameters)

    payload = {
        "service_id": KEB_SERVICE_ID,
        "context": {
        },
        "parameters": parameters
    }

    url = f"{KEB_BASE_URL}/oauth/v2/service_instances/{instance_id}?accepts_incomplete=true&service_id={KEB_SERVICE_ID}&plan_id={plan_id}"

    print("Updating with payload: ", payload)
    response = execute_request("PATCH", url, payload=payload)
    if response.status_code == 200 or response.status_code == 202:
        print("Update request successful.")
    else:
        print("Failed to update the instance.")
        print("Status Code:", response.status_code)
        print("Response:", response.text)


def deprovision(instance_id, plan_id):
    url = f"{KEB_BASE_URL}/oauth/v2/service_instances/{instance_id}?service_id={KEB_SERVICE_ID}&plan_id={plan_id}"
    response = execute_request("DELETE", url, payload=None)
    if response.status_code == 200 or response.status_code == 202:
        print("Deprovisioning request successful.")
    else:
        print("Failed to deprovision the instance.")
        print("Status Code:", response.status_code)
        print("Response:", response.text)


def provision(global_account_id="ga-id", instance_id="", subaccount_id="sa-id", plan="aws",
              user_id="testing@script.sap", region="", parameters={}, validate=False):
    if instance_id == "":
        instance_id = str(uuid.uuid4())

    plan_name_lower = plan.lower() or "aws"

    plan_info = _get_plan(plan_name_lower)
    plan_id = plan_info["id"]

    parameters = dict(parameters)
    if "region" not in parameters:
        parameters["region"] = region or _default_region(plan_name_lower)
    if "name" not in parameters:
        suffix = str(uuid.uuid4())[:4]
        parameters["name"] = "testing-cluster-" + suffix

    if validate:
        _validate_parameters(plan_info["create_schema"], parameters)

    payload = {
        "service_id": KEB_SERVICE_ID,
        "plan_id": plan_id,
        "context": {
            "globalaccount_id": global_account_id,
            "subaccount_id": subaccount_id,
            "user_id": user_id
        },
        "parameters": parameters
    }

    url = f"{KEB_BASE_URL}/oauth/v2/service_instances/{instance_id}?accepts_incomplete=true"
    response = execute_request(method="PUT", url=url, payload=payload)
    if response.status_code == 202:
        print("Provisioning request accepted.")
        print("Operation ID:", response.json().get("operation"))
        return Runtime(instance_id, response.json().get("operation"), plan_id, plan_name_lower)
    elif response.status_code == 200:
        print("Provisioning request successful.")
        return Runtime(instance_id, None, plan_id, plan_name_lower)
    else:
        print("Failed to provision the instance.")
        print("Status Code:", response.status_code)
        print("Response:", response.text)


def update_runtime_status(instance_id, state):
    valid_states = ["Pending", "Ready", "Terminating", "Failed"]
    if state not in valid_states:
        print(f"Invalid state: {state}")
        print(f"Valid states: {', '.join(valid_states)}")
        return

    try:
        if VERBOSE:
            print(f"Patching Runtime '{instance_id}' in namespace 'kcp-system' to state '{state}'...")
        runtimeNameResult = subprocess.run(
            ["kubectl", "get", "runtime", "-n", "kcp-system", "-l", f"kyma-project.io/instance-id={instance_id}", "-o",
             "name"], stdout=subprocess.PIPE)
        runtimeName = runtimeNameResult.stdout.decode().strip()
        patch_command = [
            "kubectl", "patch", runtimeName,
            "-n", "kcp-system",
            "--type", "merge",
            "--subresource", "status",
            "-p", f'{{"status": {{"state": "{state}"}}}}'
        ]
        subprocess.run(patch_command, check=True)
        print("Runtime status updated successfully.")
    except subprocess.CalledProcessError as e:
        print(f"Error occurred while updating runtime status: {e}")


VERBOSE = True


def execute_request(method, url, payload, headers=None):
    if headers is None:
        headers = {
            "X-Broker-API-Version": "2.14",
            "Content-Type": "application/json"
        }
    if VERBOSE:
        print(f"Executing {method} request to {url} with payload: {payload} and headers: {headers}")
    response = requests.request(method=method, url=url, headers=headers, json=payload)
    if VERBOSE:
        print(f"Received response with status code: {response.status_code} and body: {response.text}")
    return response

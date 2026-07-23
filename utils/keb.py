"""
This file need a requests library, install it with 'pip install requests --include-deps'

This file contains functions which can be used to provision/update/deprovision an instance. It is an initial (alpha) version.
"""

import requests
import subprocess
import uuid
import datetime

KEB_BASE_URL = "http://localhost:8080"
KEB_SERVICE_ID = "47c9dcbf-ff30-448e-ab36-d3bad66ba281"
VERBOSE = True

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
        return update(self.instance_id, self.plan_id, self.plan_name, parameters, validate=validate)

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
        return response.json().get("operation")
    else:
        print("Failed to update the instance.")
        print("Status Code:", response.status_code)
        print("Response:", response.text)
        return None


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
              user_id="testing@script.sap", region="", platform_region="", parameters={}, validate=False):
    if instance_id == "":
        instance_id = str(uuid.uuid4())

    plan_name_lower = plan.lower() or "aws"

    plan_info = _get_plan(plan_name_lower)
    plan_id = plan_info["id"]

    parameters = dict(parameters)
    if plan_name_lower == "trial":
        parameters.pop("region", None)
    else:
        if "region" not in parameters:
            parameters["region"] = region or _default_region(plan_name_lower)
    if "name" not in parameters:
        suffix = str(uuid.uuid4())[:4]
        parameters["name"] = "testing-cluster-" + suffix

    if validate:
        _validate_parameters(plan_info["create_schema"], parameters)

    context = {
        "globalaccount_id": global_account_id,
        "subaccount_id": subaccount_id,
        "user_id": user_id,
        "sm_operator_credentials": {
            "clientid": "clientid",
            "clientsecret": "clientsecret",
            "url": "https://service-manager.example.com",
            "sm_url": "https://service-manager.example.com",
        },
    }

    payload = {
        "service_id": KEB_SERVICE_ID,
        "plan_id": plan_id,
        "context": context,
        "parameters": parameters
    }

    path_region = f"/{platform_region}" if platform_region else ""
    url = f"{KEB_BASE_URL}/oauth{path_region}/v2/service_instances/{instance_id}?accepts_incomplete=true"
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
        # Wait for the Runtime CR to be created (KEB creates it asynchronously)
        import time
        runtimeName = ""
        for _ in range(30):
            result = subprocess.run(
                ["kubectl", "get", "runtime", "-n", "kcp-system", "-l", f"kyma-project.io/instance-id={instance_id}", "-o", "name"],
                stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            runtimeName = result.stdout.decode().strip()
            if runtimeName:
                break
            time.sleep(1)
        if not runtimeName:
            print(f"Runtime CR for instance '{instance_id}' not found after 30s, skipping patch.")
            return
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


_DEFAULT_MODULES = {
    "channel": "fast",
    "list": [
        {"name": "istio", "channel": ""},
        {"name": "api-gateway"},
        {"name": "btp-operator"},
        {"name": "nats"},
        {"name": "telemetry"},
    ],
}


_TRIAL_PLATFORM_REGIONS = ["cf-eu10", "cf-us10"]


def provision_many(n, global_account_id="ga-id", subaccount_id="github-actions-keb-integration", plan="trial", region="", platform_region="", parameters={}, validate=False):
    runtimes = []
    for i in range(n):
        params = dict(parameters)
        if "modules" not in params:
            params["modules"] = _DEFAULT_MODULES
        if plan.lower() == "trial":
            pr = _TRIAL_PLATFORM_REGIONS[0] if i < (n + 1) // 2 else _TRIAL_PLATFORM_REGIONS[1]
        else:
            pr = platform_region
        r = provision(
            global_account_id=global_account_id,
            subaccount_id=subaccount_id,
            plan=plan,
            region=region,
            platform_region=pr,
            parameters=params,
            validate=validate,
        )
        if r:
            runtimes.append(r)

    timestamp = datetime.datetime.now().strftime("%Y%m%d_%H%M%S")
    filename = f"instances_{timestamp}.txt"
    with open(filename, "w") as f:
        for r in runtimes:
            f.write(r.instance_id + "\n")
    print(f"Instance IDs written to {filename}")

    return runtimes


def monitor_instances(runtimes_or_file):
    if isinstance(runtimes_or_file, str):
        with open(runtimes_or_file) as f:
            instance_ids = f.read().strip().split()
    else:
        instance_ids = [r.instance_id for r in runtimes_or_file]
    params = [("instance_id", iid) for iid in instance_ids]
    url = f"{KEB_BASE_URL}/runtimes"
    response = requests.get(url, params=params)
    response.raise_for_status()

    data = response.json().get("data", [])
    by_id = {item["instanceID"]: item for item in data}

    succeeded, failed, in_progress = [], [], []
    for iid in instance_ids:
        item = by_id.get(iid)
        if item is None:
            in_progress.append(iid)
            continue
        state = item.get("status", {}).get("state", "")
        if state == "succeeded":
            succeeded.append(iid)
        elif state == "failed":
            failed.append(iid)
        else:
            in_progress.append(iid)

    print(f"succeeded: {len(succeeded)}, failed: {len(failed)}, in_progress: {len(in_progress)}")
    return {"succeeded": succeeded, "failed": failed, "in_progress": in_progress}


def deprovision_many(runtimes_or_file):
    if isinstance(runtimes_or_file, str):
        plan_id = _get_plan("trial")["id"]
        with open(runtimes_or_file) as f:
            for iid in f.read().strip().split():
                deprovision(iid, plan_id)
    else:
        for r in runtimes_or_file:
            r.deprovision()




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

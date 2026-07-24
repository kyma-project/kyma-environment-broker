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
        print(f"Deprovisioning accepted | instance: {instance_id}")
    else:
        print(f"Deprovisioning failed | instance: {instance_id} | status: {response.status_code} | {response.text}")


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
        operation_id = response.json().get("operation")
        print(f"Provisioning accepted | instance: {instance_id} | operation: {operation_id}")
        return Runtime(instance_id, operation_id, plan_id, plan_name_lower)
    elif response.status_code == 200:
        print(f"Provisioning accepted | instance: {instance_id}")
        return Runtime(instance_id, None, plan_id, plan_name_lower)
    else:
        print(f"Provisioning failed | instance: {instance_id} | status: {response.status_code} | {response.text}")


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
    global VERBOSE
    saved_verbose, VERBOSE = VERBOSE, False
    runtimes = []
    for i in range(n):
        params = dict(parameters)
        if "modules" not in params:
            params["modules"] = _DEFAULT_MODULES
        if plan.lower() == "trial":
            pr = _TRIAL_PLATFORM_REGIONS[i % len(_TRIAL_PLATFORM_REGIONS)]
        else:
            pr = platform_region
        print(f"[{i+1}/{n}] Provisioning...")
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
    VERBOSE = saved_verbose
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
    global VERBOSE
    saved_verbose, VERBOSE = VERBOSE, False
    if isinstance(runtimes_or_file, str):
        plan_id = _get_plan("trial")["id"]
        with open(runtimes_or_file) as f:
            ids = f.read().strip().split()
        for i, iid in enumerate(ids, 1):
            print(f"[{i}/{len(ids)}] Deprovisioning {iid}...")
            deprovision(iid, plan_id)
    else:
        for i, r in enumerate(runtimes_or_file, 1):
            print(f"[{i}/{len(runtimes_or_file)}] Deprovisioning {r.instance_id}...")
            r.deprovision()
    VERBOSE = saved_verbose




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


if __name__ == "__main__":
    import argparse
    parser = argparse.ArgumentParser(description="KEB provisioning utility")
    subparsers = parser.add_subparsers(dest="command", required=True)

    p = subparsers.add_parser("provision", help="Provision N instances")
    p.add_argument("n", type=int)
    p.add_argument("--global-account-id", default="ga-id")
    p.add_argument("--subaccount-id", default="github-actions-keb-integration")
    p.add_argument("--plan", default="trial")
    p.add_argument("--region", default="")

    m = subparsers.add_parser("monitor", help="Monitor instances from a file")
    m.add_argument("file")
    m.add_argument("--interval", type=int, default=0, help="Poll every N seconds until all done (0 = single check)")

    d = subparsers.add_parser("deprovision", help="Deprovision instances from a file")
    d.add_argument("file")

    s = subparsers.add_parser("simulate-provisioning-flow", help="Watch for Runtime CRs and simulate KIM/KLM after a delay")
    s.add_argument("--delay", type=int, default=900, help="Seconds to wait after CR creation before setting Ready (default: 900)")
    s.add_argument("--poll", type=int, default=10, help="Seconds between checks for new Runtime CRs (default: 10)")

    args = parser.parse_args()

    if args.command == "provision":
        provision_many(args.n, global_account_id=args.global_account_id, subaccount_id=args.subaccount_id, plan=args.plan, region=args.region)
    elif args.command == "monitor":
        import time
        while True:
            result = monitor_instances(args.file)
            if args.interval == 0 or not result["in_progress"]:
                break
            print(f"Polling again in {args.interval}s...")
            time.sleep(args.interval)
    elif args.command == "deprovision":
        deprovision_many(args.file)
    elif args.command == "simulate-provisioning-flow":
        import time
        from datetime import datetime, timezone
        processed = set()
        print(f"Watching for Runtime CRs (delay={args.delay}s, poll={args.poll}s)... Press Ctrl+C to stop.")
        while True:
            result = subprocess.run(
                ["kubectl", "get", "runtime", "-n", "kcp-system", "-o",
                 "jsonpath={range .items[*]}{.metadata.name},{.metadata.creationTimestamp}{'\\n'}{end}"],
                stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            for line in result.stdout.decode().strip().splitlines():
                if not line:
                    continue
                rid, ts = line.split(",", 1)
                if rid in processed:
                    continue
                created = datetime.fromisoformat(ts.replace("Z", "+00:00"))
                age = (datetime.now(timezone.utc) - created).total_seconds()
                wait = args.delay - age
                if wait > 0:
                    print(f"Runtime {rid} created {int(age)}s ago, firing in {int(wait)}s")
                else:
                    print(f"Running provisioning flow for {rid}...")
                    subprocess.run(["make", "run-provisioning-flow", f"RUNTIME_ID={rid}"], check=False)
                    processed.add(rid)
            time.sleep(args.poll)

14:39 start provisioningu 5k
14:51 koniec
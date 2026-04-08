"""
This file need a requests library, install it with 'pip install requests --include-deps'

This file contains functions which can be used to provision/update/deprovision an instance. It is an initial (alpha) version.
"""

import requests
import subprocess
import uuid

defaultAWSRegion = "eu-central-1"
defaultAzureRegion = "centralus"
defaultGCPRegion = "europe-west3"


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

    def __init__(self, instance_id, provisioning_operation_id):
        self.instance_id = instance_id
        self.provisioning_operation_id = provisioning_operation_id

    def __str__(self):
        return f"Runtime(instance_id={self.instance_id}, provisioning_operation_id={self.provisioning_operation_id})"

    def update_runtime_status(self, state):
        update_runtime_status(self.instance_id, state)

    def deprovision(self):
        deprovision(self.instance_id)

    def update(self, parameters={}):
        update(self.instance_id, parameters)

    def get_instance(self):
        url = f"http://localhost:8080/oauth/v2/service_instances/{self.instance_id}"
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


def update(instance_id, parameters={}):
    payload = {
        "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
        "context": {
        },
        "parameters": parameters
    }

    url = f"http://localhost:8080/oauth/v2/service_instances/{instance_id}?accepts_incomplete=true&service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281&plan_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281"

    print("Updating with payload: ", payload)
    response = execute_request("PATCH", url, payload=payload)
    # Handle the response
    if response.status_code == 200 or response.status_code == 202:
        print("Update request successful.")
    else:
        print("Failed to update the instance.")
        print("Status Code:", response.status_code)
        print("Response:", response.text)


def deprovision(instance_id):
    url = f"http://localhost:8080/oauth/v2/service_instances/{instance_id}?service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281&plan_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281"
    response = execute_request("DELETE", url)
    # Handle the response
    if response.status_code == 200 or response.status_code == 202:
        print("Deprovisioning request successful.")
    else:
        print("Failed to deprovision the instance.")
        print("Status Code:", response.status_code)
        print("Response:", response.text)


def provision(global_account_id="ga-id", instance_id="", subaccount_id="sa-id", plan="aws",
              user_id="testing@script.sap", region="", parameters={}):
    if instance_id == "":
        # generate uuid for instance_id
        instance_id = str(uuid.uuid4())

    plan_name_lower = plan.lower()
    if plan_name_lower == "":
        plan_name_lower = "aws"

    if plan_name_lower == "aws":
        planID = "361c511f-f939-4621-b228-d0fb79a1fe15"
        if region == "":
            region = defaultAWSRegion
    elif plan_name_lower == "azure":
        planID = "4deee563-e5ec-4731-b9b1-53b42d855f0c"
        if region == "":
            region = defaultAzureRegion
    elif plan_name_lower == "gcp":
        planID = "ca6e5357-707f-4565-bbbd-b3ab732597c6"
        if region == "":
            region = defaultGCPRegion

    if not "region" in parameters:
        parameters["region"] = region
    if not "name" in parameters:
        # generate short name for instance
        suffix = str(uuid.uuid4())[:4]
        parameters["name"] = "testing-cluster-" + suffix

    payload = {
        "service_id": "47c9dcbf-ff30-448e-ab36-d3bad66ba281",
        "plan_id": planID,
        "context": {
            "globalaccount_id": global_account_id,
            "subaccount_id": subaccount_id,
            "user_id": user_id
        },
        "parameters": parameters
    }

    url = f"http://localhost:8080/oauth/v2/service_instances/{instance_id}?accepts_incomplete=true"
    response = execute_request(method="PUT", url=url, payload=payload)
    # Handle the response
    if response.status_code == 202:
        print("Provisioning request accepted.")
        print("Operation ID:", response.json().get("operation"))
        return Runtime(instance_id, response.json().get("operation"))
    elif response.status_code == 200:
        print("Provisioning request successful.")
        return Runtime(instance_id, None)
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

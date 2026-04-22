"""
Seed script for manual keb-analytics testing.

Provisions a variety of instances with different parameters and applies
updates to some of them, so the analytics UI has meaningful data to display.

Usage:
    cd utils/
    python seed_analytics.py

Requires KEB running locally on http://localhost:8080.
After running, simulate KIM by marking each instance Ready:
    runtime.update_runtime_status("Ready")
(already called below after each provision)
"""

import sys
import os
sys.path.insert(0, os.path.dirname(__file__))

import keb

# ---------------------------------------------------------------------------
# Instance definitions — vary plan, region, machineType, autoScaler params
# ---------------------------------------------------------------------------

INSTANCES = [
    # AWS — eu-central-1, default machine, custom autoscaler
    dict(plan="aws", region="eu-central-1", parameters={
        "machineType": "m6i.xlarge",
        "autoScalerMin": 3,
        "autoScalerMax": 10,
    }),
    # AWS — us-east-1, different machine
    dict(plan="aws", region="us-east-1", parameters={
        "machineType": "m6i.2xlarge",
        "autoScalerMin": 2,
        "autoScalerMax": 5,
    }),
    # AWS — eu-central-1, with administrators
    dict(plan="aws", region="eu-central-1", parameters={
        "machineType": "m6i.xlarge",
        "administrators": ["admin1@example.com", "admin2@example.com"],
    }),
    # AWS — eu-central-1, minimal params (only region)
    dict(plan="aws", region="eu-central-1", parameters={}),
    # Azure — centralus, custom machine
    dict(plan="azure", region="centralus", parameters={
        "machineType": "Standard_D4s_v5",
        "autoScalerMin": 3,
        "autoScalerMax": 8,
    }),
    # Azure — eastus, default params
    dict(plan="azure", region="eastus", parameters={}),
    # GCP — europe-west3, custom autoscaler
    dict(plan="gcp", region="europe-west3", parameters={
        "machineType": "n2-standard-4",
        "autoScalerMin": 2,
        "autoScalerMax": 6,
    }),
    # GCP — us-central1
    dict(plan="gcp", region="us-central1", parameters={}),
]

# ---------------------------------------------------------------------------
# Updates to apply after provisioning (index → update params)
# ---------------------------------------------------------------------------

UPDATES = {
    0: {"autoScalerMax": 15},
    2: {"administrators": ["admin1@example.com", "admin2@example.com", "admin3@example.com"]},
    4: {"machineType": "Standard_D8s_v5", "autoScalerMin": 5, "autoScalerMax": 12},
    6: {"autoScalerMin": 3, "autoScalerMax": 9},
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    runtimes = []

    print("=== Provisioning instances ===")
    for i, spec in enumerate(INSTANCES):
        print(f"\n[{i+1}/{len(INSTANCES)}] plan={spec['plan']} region={spec['region']}")
        runtime = keb.provision(
            plan=spec["plan"],
            region=spec["region"],
            parameters=spec["parameters"],
        )
        if runtime is None:
            print(f"  FAILED — skipping")
            runtimes.append(None)
            continue
        print(f"  instance_id={runtime.instance_id}")
        runtimes.append(runtime)

        print(f"  Marking Ready...")
        runtime.update_runtime_status("Ready")

    print("\n=== Applying updates ===")
    for idx, params in UPDATES.items():
        runtime = runtimes[idx]
        if runtime is None:
            print(f"  [{idx}] skipped (provisioning failed)")
            continue
        print(f"\n[{idx}] instance_id={runtime.instance_id} params={params}")
        runtime.update(params)

    print("\n=== Done ===")
    print(f"Provisioned: {sum(1 for r in runtimes if r is not None)}/{len(INSTANCES)}")
    print(f"Updated:     {len(UPDATES)}")
    print("\nInstance IDs:")
    for i, r in enumerate(runtimes):
        if r:
            print(f"  [{i}] {r.instance_id}  plan={r.plan_name}")


if __name__ == "__main__":
    main()

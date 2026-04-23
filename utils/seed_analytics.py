"""
Seed script for manual keb-analytics testing.

Provisions a variety of instances with different parameters and applies
updates to some of them, so the analytics UI has meaningful data to display.

Usage:
    cd utils/
    python seed_analytics.py

Requires KEB running locally on http://localhost:8080.
"""

import sys
import os
sys.path.insert(0, os.path.dirname(__file__))

import keb

# ---------------------------------------------------------------------------
# Reusable building blocks
# ---------------------------------------------------------------------------

OIDC_CORP = {
    "list": [
        {
            "clientID": "corp-client-001",
            "issuerURL": "https://sso.corp.example.com",
            "groupsClaim": "groups",
            "groupsPrefix": "corp:",
            "usernameClaim": "email",
            "usernamePrefix": "-",
            "signingAlgs": ["RS256"],
        }
    ]
}

OIDC_DEV = {
    "list": [
        {
            "clientID": "dev-client-abc",
            "issuerURL": "https://dev.idp.example.com",
            "groupsClaim": "roles",
            "groupsPrefix": "-",
            "usernameClaim": "sub",
            "usernamePrefix": "dev:",
            "signingAlgs": ["RS256", "ES256"],
        }
    ]
}

OIDC_MULTI = {
    "list": [
        {
            "clientID": "primary-client",
            "issuerURL": "https://primary.idp.example.com",
            "groupsClaim": "groups",
            "groupsPrefix": "-",
            "usernameClaim": "email",
            "usernamePrefix": "-",
            "signingAlgs": ["RS256"],
        },
        {
            "clientID": "secondary-client",
            "issuerURL": "https://secondary.idp.example.com",
            "groupsClaim": "groups",
            "groupsPrefix": "secondary:",
            "usernameClaim": "sub",
            "usernamePrefix": "sec:",
            "signingAlgs": ["RS256", "ES256"],
        },
    ]
}

OIDC_TRIPLE = {
    "list": [
        {
            "clientID": "idp-a-client",
            "issuerURL": "https://idp-a.example.com",
            "groupsClaim": "groups",
            "groupsPrefix": "-",
            "usernameClaim": "email",
            "usernamePrefix": "-",
            "signingAlgs": ["RS256"],
        },
        {
            "clientID": "idp-b-client",
            "issuerURL": "https://idp-b.example.com",
            "groupsClaim": "teams",
            "groupsPrefix": "b:",
            "usernameClaim": "sub",
            "usernamePrefix": "b:",
            "signingAlgs": ["ES256"],
        },
        {
            "clientID": "idp-c-client",
            "issuerURL": "https://idp-c.example.com",
            "groupsClaim": "groups",
            "groupsPrefix": "c:",
            "usernameClaim": "email",
            "usernamePrefix": "-",
            "signingAlgs": ["RS256"],
            "requiredClaims": ["env=production"],
        },
    ]
}

ADMINS_SMALL = ["alice@example.com", "bob@example.com"]
ADMINS_MEDIUM = ["alice@example.com", "bob@example.com", "carol@example.com", "dave@example.com"]
ADMINS_LARGE = [
    "alice@example.com", "bob@example.com", "carol@example.com",
    "dave@example.com", "eve@example.com", "frank@example.com",
    "grace@example.com",
]

# ---------------------------------------------------------------------------
# Instance definitions (20 instances)
# ---------------------------------------------------------------------------

INSTANCES = [
    # 0 — AWS, eu-central-1, minimal
    dict(plan="aws", region="eu-central-1", parameters={}),

    # 1 — AWS, eu-central-1, custom machine + autoscaler
    dict(plan="aws", region="eu-central-1", parameters={
        "machineType": "m6i.xlarge",
        "autoScalerMin": 3,
        "autoScalerMax": 10,
    }),

    # 2 — AWS, eu-central-1, corp OIDC + small admins
    dict(plan="aws", region="eu-central-1", parameters={
        "machineType": "m6i.2xlarge",
        "autoScalerMin": 3,
        "autoScalerMax": 8,
        "oidc": OIDC_CORP,
        "administrators": ADMINS_SMALL,
    }),

    # 3 — AWS, us-east-1, multi OIDC + medium admins + additional worker pool
    dict(plan="aws", region="us-east-1", parameters={
        "machineType": "m6i.xlarge",
        "autoScalerMin": 3,
        "autoScalerMax": 12,
        "oidc": OIDC_MULTI,
        "administrators": ADMINS_MEDIUM,
        "additionalWorkerNodePools": [
            {"name": "gpu-pool", "machineType": "g4dn.xlarge", "haZones": True, "autoScalerMin": 3, "autoScalerMax": 6},
        ],
    }),

    # 4 — AWS, us-east-1, triple OIDC + large admins + two worker pools
    dict(plan="aws", region="us-east-1", parameters={
        "machineType": "m6i.4xlarge",
        "autoScalerMin": 3,
        "autoScalerMax": 15,
        "oidc": OIDC_TRIPLE,
        "administrators": ADMINS_LARGE,
        "additionalWorkerNodePools": [
            {"name": "compute-pool", "machineType": "c7i.2xlarge", "haZones": True, "autoScalerMin": 3, "autoScalerMax": 9},
            {"name": "mem-pool", "machineType": "m6i.4xlarge", "haZones": True, "autoScalerMin": 3, "autoScalerMax": 6},
        ],
    }),

    # 5 — AWS, eu-west-1, dev OIDC
    dict(plan="aws", region="eu-west-1", parameters={
        "machineType": "m5.xlarge",
        "autoScalerMin": 3,
        "autoScalerMax": 6,
        "oidc": OIDC_DEV,
    }),

    # 6 — AWS, us-west-2, large admins
    dict(plan="aws", region="us-west-2", parameters={
        "machineType": "m6i.2xlarge",
        "autoScalerMin": 3,
        "autoScalerMax": 10,
        "administrators": ADMINS_LARGE,
    }),

    # 7 — Azure, eastus, minimal
    dict(plan="azure", region="eastus", parameters={}),

    # 8 — Azure, eastus, custom machine + autoscaler
    dict(plan="azure", region="eastus", parameters={
        "machineType": "Standard_D4s_v5",
        "autoScalerMin": 3,
        "autoScalerMax": 8,
    }),

    # 9 — Azure, centralus, corp OIDC + medium admins
    dict(plan="azure", region="centralus", parameters={
        "machineType": "Standard_D8s_v5",
        "autoScalerMin": 3,
        "autoScalerMax": 10,
        "oidc": OIDC_CORP,
        "administrators": ADMINS_MEDIUM,
    }),

    # 10 — Azure, westeurope, multi OIDC + additional worker pool
    dict(plan="azure", region="westeurope", parameters={
        "machineType": "Standard_D4s_v5",
        "autoScalerMin": 3,
        "autoScalerMax": 12,
        "oidc": OIDC_MULTI,
        "additionalWorkerNodePools": [
            {"name": "batch-pool", "machineType": "Standard_D8s_v5", "haZones": True, "autoScalerMin": 3, "autoScalerMax": 5},
        ],
    }),

    # 11 — Azure, northeurope, triple OIDC + large admins + two worker pools
    dict(plan="azure", region="northeurope", parameters={
        "machineType": "Standard_D16s_v5",
        "autoScalerMin": 3,
        "autoScalerMax": 20,
        "oidc": OIDC_TRIPLE,
        "administrators": ADMINS_LARGE,
        "additionalWorkerNodePools": [
            {"name": "gpu-pool", "machineType": "Standard_NC4as_T4_v3", "haZones": True, "autoScalerMin": 3, "autoScalerMax": 6},
            {"name": "spot-pool", "machineType": "Standard_D8s_v5", "haZones": False, "autoScalerMin": 0, "autoScalerMax": 1},
        ],
    }),

    # 12 — Azure, uksouth, dev OIDC + small admins
    dict(plan="azure", region="uksouth", parameters={
        "machineType": "Standard_D4s_v5",
        "autoScalerMin": 3,
        "autoScalerMax": 6,
        "oidc": OIDC_DEV,
        "administrators": ADMINS_SMALL,
    }),

    # 13 — GCP, europe-west3, minimal
    dict(plan="gcp", region="europe-west3", parameters={}),

    # 14 — GCP, europe-west3, custom machine + autoscaler
    dict(plan="gcp", region="europe-west3", parameters={
        "machineType": "n2-standard-4",
        "autoScalerMin": 3,
        "autoScalerMax": 9,
    }),

    # 15 — GCP, us-central1, corp OIDC + medium admins
    dict(plan="gcp", region="us-central1", parameters={
        "machineType": "n2-standard-8",
        "autoScalerMin": 3,
        "autoScalerMax": 12,
        "oidc": OIDC_CORP,
        "administrators": ADMINS_MEDIUM,
    }),

    # 16 — GCP, us-central1, multi OIDC + additional worker pool
    dict(plan="gcp", region="us-central1", parameters={
        "machineType": "n2-standard-4",
        "autoScalerMin": 3,
        "autoScalerMax": 10,
        "oidc": OIDC_MULTI,
        "additionalWorkerNodePools": [
            {"name": "ml-pool", "machineType": "n2-standard-8", "haZones": True, "autoScalerMin": 3, "autoScalerMax": 6},
        ],
    }),

    # 17 — GCP, asia-south1, triple OIDC + large admins + two worker pools
    dict(plan="gcp", region="asia-south1", parameters={
        "machineType": "n2-standard-16",
        "autoScalerMin": 3,
        "autoScalerMax": 18,
        "oidc": OIDC_TRIPLE,
        "administrators": ADMINS_LARGE,
        "additionalWorkerNodePools": [
            {"name": "gpu-pool", "machineType": "n2-standard-8", "haZones": True, "autoScalerMin": 3, "autoScalerMax": 6},
            {"name": "batch-pool", "machineType": "n2-standard-4", "haZones": False, "autoScalerMin": 0, "autoScalerMax": 1},
        ],
    }),

    # 18 — GCP, europe-west4, dev OIDC
    dict(plan="gcp", region="europe-west4", parameters={
        "machineType": "n2-standard-4",
        "autoScalerMin": 3,
        "autoScalerMax": 6,
        "oidc": OIDC_DEV,
    }),

    # 19 — GCP, asia-northeast1, large admins + worker pool
    dict(plan="gcp", region="asia-northeast1", parameters={
        "machineType": "n2-standard-8",
        "autoScalerMin": 3,
        "autoScalerMax": 15,
        "administrators": ADMINS_LARGE,
        "additionalWorkerNodePools": [
            {"name": "infra-pool", "machineType": "n2-standard-4", "haZones": True, "autoScalerMin": 3, "autoScalerMax": 5},
        ],
    }),
]

# ---------------------------------------------------------------------------
# Updates to apply after provisioning (index → update params)
# ---------------------------------------------------------------------------

UPDATES = {
    1:  {"autoScalerMax": 15},
    2:  {"administrators": ADMINS_MEDIUM},
    4:  {"autoScalerMax": 20},
    6:  {"administrators": ADMINS_SMALL},
    8:  {"machineType": "Standard_D8s_v5", "autoScalerMin": 5, "autoScalerMax": 12},
    9:  {"administrators": ADMINS_LARGE},
    14: {"autoScalerMin": 3, "autoScalerMax": 12},
    15: {"administrators": ADMINS_LARGE, "autoScalerMax": 15},
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

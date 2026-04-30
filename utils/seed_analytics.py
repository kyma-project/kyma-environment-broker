"""
Seed script for manual keb-analytics testing.

Provisions 1000 instances with varied parameters across all major plans and
regions, then applies updates to ~40% of them.

Usage:
    cd utils/
    python seed_analytics.py [--count N] [--skip-updates]

Requires KEB running locally on http://localhost:8080.
"""

import sys
import os
import random
import argparse
import time
import requests

sys.path.insert(0, os.path.dirname(__file__))
import keb

keb.VERBOSE = False

# ---------------------------------------------------------------------------
# Parameter pools
# ---------------------------------------------------------------------------

OIDC_CONFIGS = [
    None,  # no OIDC (~40% of instances)
    {
        "list": [{
            "clientID": "corp-client-001",
            "issuerURL": "https://sso.corp.example.com",
            "groupsClaim": "groups",
            "groupsPrefix": "corp:",
            "usernameClaim": "email",
            "usernamePrefix": "-",
            "signingAlgs": ["RS256"],
        }]
    },
    {
        "list": [{
            "clientID": "dev-client-abc",
            "issuerURL": "https://dev.idp.example.com",
            "groupsClaim": "roles",
            "groupsPrefix": "-",
            "usernameClaim": "sub",
            "usernamePrefix": "dev:",
            "signingAlgs": ["RS256", "ES256"],
        }]
    },
    {
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
    },
    {
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
    },
]

ADMIN_POOLS = [
    [],  # no explicit admins
    ["alice@example.com", "bob@example.com"],
    ["alice@example.com", "bob@example.com", "carol@example.com", "dave@example.com"],
    [
        "alice@example.com", "bob@example.com", "carol@example.com",
        "dave@example.com", "eve@example.com", "frank@example.com",
        "grace@example.com",
    ],
]

WORKER_POOLS = {
    "aws": [
        {"name": "gpu-pool",     "machineType": "g4dn.xlarge",  "haZones": True,  "autoScalerMin": 3, "autoScalerMax": 6},
        {"name": "compute-pool", "machineType": "c7i.2xlarge",   "haZones": True,  "autoScalerMin": 3, "autoScalerMax": 9},
        {"name": "mem-pool",     "machineType": "m6i.4xlarge",   "haZones": True,  "autoScalerMin": 3, "autoScalerMax": 6},
        {"name": "spot-pool",    "machineType": "m5.xlarge",     "haZones": False, "autoScalerMin": 0, "autoScalerMax": 5},
    ],
    "azure": [
        {"name": "gpu-pool",   "machineType": "Standard_NC4as_T4_v3", "haZones": True,  "autoScalerMin": 3, "autoScalerMax": 6},
        {"name": "batch-pool", "machineType": "Standard_D8s_v5",      "haZones": True,  "autoScalerMin": 3, "autoScalerMax": 5},
        {"name": "spot-pool",  "machineType": "Standard_D4s_v5",      "haZones": False, "autoScalerMin": 0, "autoScalerMax": 8},
    ],
    "gcp": [
        {"name": "ml-pool",    "machineType": "n2-standard-8", "haZones": True,  "autoScalerMin": 3, "autoScalerMax": 6},
        {"name": "gpu-pool",   "machineType": "n2-standard-8", "haZones": True,  "autoScalerMin": 3, "autoScalerMax": 4},
        {"name": "batch-pool", "machineType": "n2-standard-4", "haZones": False, "autoScalerMin": 0, "autoScalerMax": 3},
    ],
}

# Plan → regions with realistic distribution weights (common regions heavier)
PLAN_REGIONS = {
    "aws": [
        ("eu-central-1", 25), ("us-east-1", 22), ("eu-west-2", 12),
        ("us-west-2", 12), ("ap-southeast-1", 10), ("ap-northeast-1", 8),
        ("ca-central-1", 6), ("ap-south-1", 5),
    ],
    "azure": [
        ("eastus", 22), ("westeurope", 18), ("northeurope", 14),
        ("centralus", 10), ("uksouth", 8), ("southeastasia", 7),
        ("japaneast", 6), ("australiaeast", 5), ("switzerlandnorth", 5),
        ("brazilsouth", 3), ("canadacentral", 2),
    ],
    "gcp": [
        ("europe-west3", 30), ("us-central1", 25), ("us-east4", 15),
        ("europe-west4", 12), ("asia-south1", 10), ("asia-northeast1", 8),
    ],
    "azure_lite": [
        ("eastus", 30), ("westeurope", 25), ("northeurope", 20),
        ("centralus", 15), ("uksouth", 10),
    ],
    "trial": [
        ("eu-central-1", 50), ("us-east-1", 30), ("eu-west-2", 20),
    ],
}

MACHINE_TYPES = {
    "aws":        ["m6i.large", "m6i.xlarge", "m6i.2xlarge", "m6i.4xlarge", "m5.xlarge"],
    "azure":      ["Standard_D2s_v5", "Standard_D4s_v5", "Standard_D8s_v5", "Standard_D16s_v5", "Standard_D4_v3", "Standard_D8_v3"],
    "gcp":        ["n2-standard-2", "n2-standard-4", "n2-standard-8", "n2-standard-16"],
    "azure_lite": ["Standard_D2s_v5", "Standard_D4s_v5"],
    "trial":      ["m5.xlarge", "Standard_D4s_v5", "n2-standard-2"],
}

# Plan distribution: (plan_name, weight) — only plans available in local catalog
PLAN_WEIGHTS = [
    ("aws",        45),
    ("azure",      33),
    ("gcp",        14),
    ("azure_lite",  5),
    ("trial",       3),
]


def weighted_choice(choices):
    """Pick an item from a list of (value, weight) tuples."""
    items, weights = zip(*choices)
    return random.choices(items, weights=weights, k=1)[0]


def build_parameters(plan, rng):
    """Generate randomised provisioning parameters for a given plan."""
    params = {}

    machines = MACHINE_TYPES.get(plan, ["m6i.large"])
    # ~30% of instances use default machine type (no explicit machineType)
    if rng.random() > 0.30:
        params["machineType"] = rng.choice(machines)

    # autoScaler — set for ~70% of instances
    if rng.random() > 0.30:
        min_val = rng.choice([3, 3, 3, 5, 5, 10])
        max_val = rng.choice([6, 8, 10, 12, 15, 20, 20])
        max_val = max(max_val, min_val + 1)
        params["autoScalerMin"] = min_val
        params["autoScalerMax"] = max_val

    # OIDC — set for ~60% of instances (skip None from the pool)
    oidc_pool_with_weights = [(None, 40)] + [(o, 15) for o in OIDC_CONFIGS[1:]]
    oidc = weighted_choice(oidc_pool_with_weights)
    if oidc is not None:
        params["oidc"] = oidc

    # admins — set for ~50% of instances
    admin_pool_with_weights = [([], 50), (ADMIN_POOLS[1], 20), (ADMIN_POOLS[2], 20), (ADMIN_POOLS[3], 10)]
    admins = weighted_choice(admin_pool_with_weights)
    if admins:
        params["administrators"] = admins

    # additional worker pools — set for ~25% of non-trial/free instances
    if plan in WORKER_POOLS and rng.random() < 0.25:
        pool_count = rng.choice([1, 1, 2])
        pools = rng.sample(WORKER_POOLS[plan], min(pool_count, len(WORKER_POOLS[plan])))
        params["additionalWorkerNodePools"] = pools

    return params


def poll_until_done(runtimes, label, poll_interval=2, timeout=300):
    """Poll last_operation for each runtime until all reach a terminal state (succeeded/failed)."""
    pending = {
        r.instance_id: r
        for r in runtimes
        if r is not None and r.provisioning_operation_id is not None
    }
    if not pending:
        return

    headers = {"X-Broker-API-Version": "2.14"}
    succeeded = failed = 0
    deadline = time.time() + timeout
    last_print = time.time()

    print(f"\n=== Waiting for {len(pending)} {label} operations to complete (timeout={timeout}s) ===")

    while pending and time.time() < deadline:
        done = []
        for instance_id, runtime in pending.items():
            url = (
                f"{keb.KEB_BASE_URL}/oauth/v2/service_instances/{instance_id}"
                f"/last_operation?operation={runtime.provisioning_operation_id}"
            )
            try:
                resp = requests.get(url, headers=headers, timeout=5)
                if resp.status_code == 200:
                    state = resp.json().get("state", "")
                    if state == "succeeded":
                        succeeded += 1
                        done.append(instance_id)
                    elif state == "failed":
                        failed += 1
                        done.append(instance_id)
            except requests.RequestException:
                pass
        for iid in done:
            del pending[iid]

        now = time.time()
        if now - last_print >= 10 or not pending:
            total = succeeded + failed + len(pending)
            print(f"  succeeded={succeeded}  failed={failed}  pending={len(pending)}  total={total}")
            last_print = now

        if pending:
            time.sleep(poll_interval)

    if pending:
        print(f"  WARNING: {len(pending)} operations still pending after timeout")
    print(f"  Final: succeeded={succeeded}  failed={failed}  timed_out={len(pending)}")


def build_update_parameters(plan, rng):
    """Generate randomised update parameters."""
    updates = {}

    machines = MACHINE_TYPES.get(plan, ["m6i.large"])
    if rng.random() < 0.40:
        updates["machineType"] = rng.choice(machines)

    if rng.random() < 0.50:
        min_val = rng.choice([3, 5, 5, 10])
        max_val = rng.choice([8, 10, 12, 15, 20])
        max_val = max(max_val, min_val + 1)
        updates["autoScalerMin"] = min_val
        updates["autoScalerMax"] = max_val

    if rng.random() < 0.30:
        updates["administrators"] = rng.choice(ADMIN_POOLS[1:])

    if rng.random() < 0.20 and plan in WORKER_POOLS:
        pool_count = rng.choice([1, 2])
        pools = rng.sample(WORKER_POOLS[plan], min(pool_count, len(WORKER_POOLS[plan])))
        updates["additionalWorkerNodePools"] = pools

    return updates


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    parser = argparse.ArgumentParser(description="Seed keb-analytics with test data.")
    parser.add_argument("--count", type=int, default=1000, help="Number of instances to provision (default: 1000)")
    parser.add_argument("--seed",  type=int, default=42,   help="Random seed for reproducibility (default: 42)")
    parser.add_argument("--skip-updates", action="store_true", help="Skip the update phase")
    parser.add_argument("--poll-timeout", type=int, default=600, help="Seconds to wait for operations to complete (default: 600)")
    args = parser.parse_args()

    rng = random.Random(args.seed)
    count = args.count

    print(f"Seeding {count} instances (random seed={args.seed})...")

    runtimes = []

    print("\n=== Provisioning instances ===")
    for i in range(count):
        plan = weighted_choice(PLAN_WEIGHTS)
        region = weighted_choice(PLAN_REGIONS[plan])
        parameters = build_parameters(plan, rng)

        if (i + 1) % 100 == 0 or i == 0:
            print(f"  [{i+1}/{count}] plan={plan} region={region}")

        runtime = keb.provision(plan=plan, region=region, parameters=parameters)
        if runtime is None:
            runtimes.append(None)
            continue

        runtime.update_runtime_status("Ready")
        runtimes.append(runtime)

    provisioned = sum(1 for r in runtimes if r is not None)
    print(f"\nProvisioned: {provisioned}/{count}")

    poll_until_done(runtimes, "provisioning", timeout=args.poll_timeout)

    if args.skip_updates:
        print("\n=== Updates skipped ===")
        print("\n=== Done ===")
        return

    # Apply updates to ~40% of successfully provisioned instances
    update_targets = [r for r in runtimes if r is not None]
    rng.shuffle(update_targets)
    update_targets = update_targets[:int(len(update_targets) * 0.40)]

    print(f"\n=== Applying updates to {len(update_targets)} instances ===")
    for i, runtime in enumerate(update_targets):
        params = build_update_parameters(runtime.plan_name, rng)
        if not params:
            continue
        if (i + 1) % 100 == 0 or i == 0:
            print(f"  [{i+1}/{len(update_targets)}] instance_id={runtime.instance_id}")
        op_id = runtime.update(params)
        if op_id:
            runtime.provisioning_operation_id = op_id

    poll_until_done(update_targets, "update", timeout=args.poll_timeout)

    print(f"\n=== Done ===")
    print(f"Provisioned: {provisioned}/{count}")
    print(f"Updated:     {len(update_targets)}")


if __name__ == "__main__":
    main()

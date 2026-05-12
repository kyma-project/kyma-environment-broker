"""
Send update operations to existing KEB instances for analytics testing.

Fetches active instance IDs + service_plan_id from postgres, then sends
PATCH /oauth/v2/service_instances/<id> to KEB for ~40% of them.

Usage:
    cd utils/
    python seed_updates.py [--fraction 0.4] [--seed 42]
                           [--db-host HOST] [--db-port PORT]
                           [--db-name NAME] [--db-user USER] [--db-password PWD]
"""

import sys
import os
import random
import argparse
import time

import psycopg2

sys.path.insert(0, os.path.dirname(__file__))
import keb

keb.VERBOSE = False

MACHINE_TYPES = {
    "aws":        ["m6i.large", "m6i.xlarge", "m6i.2xlarge", "m6i.4xlarge", "m5.xlarge"],
    "azure":      ["Standard_D2s_v5", "Standard_D4s_v5", "Standard_D8s_v5", "Standard_D16s_v5"],
    "gcp":        ["n2-standard-2", "n2-standard-4", "n2-standard-8", "n2-standard-16"],
    "azure_lite": ["Standard_D2s_v5", "Standard_D4s_v5"],
    "trial":      ["m5.xlarge"],
}

# plan UUID -> short name (from broker.PlanIDsMapping)
PLAN_ID_TO_NAME = {
    "361c511f-f939-4621-b228-d0fb79a1fe15": "aws",
    "4deee563-e5ec-4731-b9b1-53b42d855f0c": "azure",
    "ca6e5357-707f-4565-bbbd-e3aceee8a579": "gcp",
    "8cb22518-aa26-44c5-91a0-e669ec9bf443": "azure_lite",
    "7d55d31d-35ae-4438-bf13-6ffdfa107d9f": "trial",
    "b1a5764e-2ea1-4f6c-a17f-fc01c56abfd3": "aws",   # aws_ha
    "03b812b9-a256-4d03-b852-850b52d952d2": "azure",  # azure_ha
}

ACL_POOLS = [
    {"allowedCIDRs": ["10.0.0.0/8"]},
    {"allowedCIDRs": ["10.0.0.0/8", "172.16.0.0/12"]},
    {"allowedCIDRs": ["10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"]},
]


def build_update_params(plan_name, rng):
    params = {}
    machines = MACHINE_TYPES.get(plan_name, ["m6i.large"])
    if rng.random() < 0.60:
        params["machineType"] = rng.choice(machines)
    if rng.random() < 0.50:
        min_val = rng.choice([3, 3, 5, 5, 10])
        max_val = rng.choice([6, 8, 10, 12, 15, 20])
        max_val = max(max_val, min_val + 1)
        params["autoScalerMin"] = min_val
        params["autoScalerMax"] = max_val
    if rng.random() < 0.20:
        params["accessControlList"] = rng.choice(ACL_POOLS)
    if rng.random() < 0.10 and plan_name in ("aws", "gcp"):
        params["gvisor"] = {"enabled": rng.choice([True, False])}
    return params


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--fraction", type=float, default=0.4,
                        help="Fraction of instances to update (default: 0.4)")
    parser.add_argument("--seed", type=int, default=42)
    parser.add_argument("--db-host",     default="localhost")
    parser.add_argument("--db-port",     type=int, default=5432)
    parser.add_argument("--db-name",     default="postgresdb")
    parser.add_argument("--db-user",     default="postgresadmin")
    parser.add_argument("--db-password", default="password")
    args = parser.parse_args()

    db_dsn = (f"host={args.db_host} port={args.db_port} dbname={args.db_name} "
              f"user={args.db_user} password={args.db_password}")

    rng = random.Random(args.seed)

    conn = psycopg2.connect(db_dsn)
    cur = conn.cursor()
    cur.execute("""
        SELECT instance_id, service_plan_id
        FROM instances
        WHERE deleted_at = '0001-01-01 00:00:00+00'
        ORDER BY instance_id
    """)
    rows = cur.fetchall()
    conn.close()

    rng.shuffle(rows)
    targets = rows[:int(len(rows) * args.fraction)]
    print(f"Sending updates to {len(targets)}/{len(rows)} instances...")

    pending_ops = []
    for i, (instance_id, plan_id) in enumerate(targets):
        plan_name = PLAN_ID_TO_NAME.get(plan_id, "aws")
        params = build_update_params(plan_name, rng)
        if not params:
            continue

        if (i + 1) % 50 == 0 or i == 0:
            print(f"  [{i+1}/{len(targets)}]")

        try:
            op_id = keb.update(instance_id, plan_id, plan_name, params)
            if op_id:
                pending_ops.append((instance_id, op_id))
        except Exception:
            pass

    print(f"Queued {len(pending_ops)} update operations. Polling for completion...")

    timeout = 600
    deadline = time.time() + timeout
    while time.time() < deadline and pending_ops:
        time.sleep(10)
        conn = psycopg2.connect(db_dsn)
        cur = conn.cursor()
        cur.execute("""
            SELECT instance_id, state FROM operations
            WHERE type='update' AND state IN ('succeeded','failed')
        """)
        done = {row[0] for row in cur.fetchall()}
        conn.close()
        pending_ops = [(iid, oid) for iid, oid in pending_ops if iid not in done]
        print(f"  pending: {len(pending_ops)}")
        if not pending_ops:
            break

    conn = psycopg2.connect(db_dsn)
    cur = conn.cursor()
    cur.execute("SELECT count(*) FROM operations WHERE type='update' AND state='succeeded'")
    count = cur.fetchone()[0]
    conn.close()
    print(f"Done. {count} succeeded update operations in DB.")


if __name__ == "__main__":
    main()

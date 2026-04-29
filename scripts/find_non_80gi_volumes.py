#!/usr/bin/env python3
"""
Find instances where any worker or additionalWorker has a volume size != 80Gi.
Usage:
    kcp rt -ojson --runtime-config | python3 find_non_80gi_volumes.py
    python3 find_non_80gi_volumes.py < output.json
    python3 find_non_80gi_volumes.py output.json
"""

import json
import sys

EXPECTED_SIZE = "80Gi"


def check_volumes(runtime):
    instance_id = runtime.get("instanceID", "")
    runtime_id = runtime.get("runtimeID", "")

    provider = (
        runtime.get("runtimeConfig", {})
        .get("spec", {})
        .get("shoot", {})
        .get("provider", {})
    )

    bad_workers = []
    for pool_key in ("workers", "additionalWorkers"):
        for worker in provider.get(pool_key, []):
            size = worker.get("volume", {}).get("size")
            if size and size != EXPECTED_SIZE:
                bad_workers.append({
                    "pool": pool_key,
                    "name": worker.get("name", ""),
                    "size": size,
                })

    if bad_workers:
        return {"instanceID": instance_id, "runtimeID": runtime_id, "badWorkers": bad_workers}
    return None


def main():
    if len(sys.argv) > 1:
        with open(sys.argv[1]) as f:
            data = json.load(f)
    else:
        data = json.load(sys.stdin)

    # Support both wrapped {"data": [...]} and bare [...] responses
    runtimes = data if isinstance(data, list) else data.get("data", [])

    results = [r for r in (check_volumes(rt) for rt in runtimes) if r]

    if not results:
        print("All workers have 80Gi volumes.")
        return

    print(f"Found {len(results)} instance(s) with non-{EXPECTED_SIZE} volumes:\n")
    for r in results:
        print(f"  instanceID : {r['instanceID']}")
        print(f"  runtimeID  : {r['runtimeID']}")
        for w in r["badWorkers"]:
            print(f"    [{w['pool']}] worker={w['name']}  size={w['size']}")
        print()


if __name__ == "__main__":
    main()

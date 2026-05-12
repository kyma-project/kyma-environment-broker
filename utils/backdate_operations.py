"""
Backdate created_at timestamps of provisioning operations and instances
to simulate data spread over a past time window.

Optionally simulate parameters introduced at specific points in time using
--param-cutoff PARAM:DAYS_AGO. Instances whose backdated created_at falls on
or after the cutoff have that parameter set to true; instances before the
cutoff have it stripped. Multiple --param-cutoff pairs can be specified.

Usage:
    python3 backdate_operations.py [--days N] [--db-host HOST] [--db-port PORT]
                                   [--db-name NAME] [--db-user USER] [--db-password PWD]
                                   [--param-cutoff PARAM:DAYS_AGO ...]

Example:
    python3 backdate_operations.py --days 90 --param-cutoff ingressFiltering:30

Defaults match the local k3d KEB setup (port-forwarded postgres on localhost:5432).

Requires: psycopg2  (pip install psycopg2-binary)
"""

import argparse
import json
import random
import psycopg2


def backdate(conn, days, param_cutoffs):
    with conn.cursor() as cur:
        # Fetch all provision operation IDs, instance IDs and current provisioning_parameters
        cur.execute("""
            SELECT id, instance_id, provisioning_parameters FROM operations
            WHERE type = 'provision'
            ORDER BY created_at
        """)
        rows = cur.fetchall()
        if not rows:
            print("No provisioning operations found.")
            return

        print(f"Backdating {len(rows)} provisioning operations over the past {days} days...")

        for op_id, instance_id, prov_params_raw in rows:
            offset_seconds = random.randint(0, days * 86400)

            cur.execute("""
                UPDATE operations
                SET created_at = NOW() - make_interval(secs => %s)
                WHERE id = %s
            """, (offset_seconds, op_id))
            cur.execute("""
                UPDATE instances
                SET created_at = NOW() - make_interval(secs => %s)
                WHERE instance_id = %s
            """, (offset_seconds, instance_id))

            if param_cutoffs:
                try:
                    if isinstance(prov_params_raw, dict):
                        params = prov_params_raw
                    else:
                        params = json.loads(prov_params_raw) if prov_params_raw else {}
                except (json.JSONDecodeError, TypeError):
                    continue

                changed = False
                for param, cutoff_days in param_cutoffs.items():
                    after_cutoff = offset_seconds <= cutoff_days * 86400
                    if after_cutoff:
                        params.setdefault("parameters", {})[param] = True
                        changed = True
                    else:
                        if params.get("parameters", {}).pop(param, None) is not None:
                            changed = True

                if changed:
                    cur.execute("""
                        UPDATE operations
                        SET provisioning_parameters = %s
                        WHERE id = %s
                    """, (json.dumps(params), op_id))

        conn.commit()

        for param, cutoff_days in param_cutoffs.items():
            print(f"  {param}: set for instances provisioned within the last {cutoff_days} days (~{cutoff_days/days*100:.0f}%)")
        print(f"Done. Timestamps spread randomly over the past {days} days.")


def main():
    parser = argparse.ArgumentParser(description="Backdate KEB operation timestamps for analytics testing.")
    parser.add_argument("--days",        type=int, default=90,            help="Spread timestamps over this many past days (default: 90)")
    parser.add_argument("--db-host",     default="localhost",              help="DB host (default: localhost)")
    parser.add_argument("--db-port",     type=int, default=5432,          help="DB port (default: 5432)")
    parser.add_argument("--db-name",     default="postgresdb",            help="DB name (default: postgresdb)")
    parser.add_argument("--db-user",     default="postgresadmin",         help="DB user (default: postgresadmin)")
    parser.add_argument("--db-password", default="admin12345678901#",     help="DB password (default: admin12345678901#)")
    parser.add_argument("--param-cutoff", action="append", default=[], metavar="PARAM:DAYS_AGO",
                        dest="param_cutoff",
                        help="Simulate a parameter introduced DAYS_AGO days ago. "
                             "Instances backdated on or after the cutoff get the parameter set to true; "
                             "earlier instances have it stripped. Can be specified multiple times.")
    args = parser.parse_args()

    param_cutoffs = {}
    for entry in args.param_cutoff:
        if ":" not in entry:
            parser.error(f"--param-cutoff must be PARAM:DAYS_AGO, got: {entry!r}")
        param, _, days_str = entry.partition(":")
        try:
            cutoff_days = int(days_str)
        except ValueError:
            parser.error(f"--param-cutoff days must be an integer, got: {days_str!r}")
        param_cutoffs[param.strip()] = cutoff_days

    conn = psycopg2.connect(
        host=args.db_host,
        port=args.db_port,
        dbname=args.db_name,
        user=args.db_user,
        password=args.db_password,
    )
    try:
        backdate(conn, args.days, param_cutoffs)
    finally:
        conn.close()


if __name__ == "__main__":
    main()

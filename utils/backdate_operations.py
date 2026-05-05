"""
Backdate created_at timestamps of provisioning operations and instances
to simulate data spread over a past time window.

Usage:
    python3 backdate_operations.py [--days N] [--db-host HOST] [--db-port PORT]
                                   [--db-name NAME] [--db-user USER] [--db-password PWD]

Defaults match the local k3d KEB setup (port-forwarded postgres on localhost:5432).

Requires: psycopg2  (pip install psycopg2-binary)
"""

import argparse
import random
import psycopg2


def backdate(conn, days):
    with conn.cursor() as cur:
        # Fetch all provision operation IDs and their instance IDs
        cur.execute("""
            SELECT id, instance_id FROM operations
            WHERE type = 'provision'
            ORDER BY created_at
        """)
        rows = cur.fetchall()
        if not rows:
            print("No provisioning operations found.")
            return

        print(f"Backdating {len(rows)} provisioning operations over the past {days} days...")

        for op_id, instance_id in rows:
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

        conn.commit()
        print(f"Done. Timestamps spread randomly over the past {days} days.")


def main():
    parser = argparse.ArgumentParser(description="Backdate KEB operation timestamps for analytics testing.")
    parser.add_argument("--days",        type=int, default=90,            help="Spread timestamps over this many past days (default: 90)")
    parser.add_argument("--db-host",     default="localhost",              help="DB host (default: localhost)")
    parser.add_argument("--db-port",     type=int, default=5432,          help="DB port (default: 5432)")
    parser.add_argument("--db-name",     default="postgresdb",            help="DB name (default: postgresdb)")
    parser.add_argument("--db-user",     default="postgresadmin",         help="DB user (default: postgresadmin)")
    parser.add_argument("--db-password", default="password",              help="DB password (default: password)")
    args = parser.parse_args()

    conn = psycopg2.connect(
        host=args.db_host,
        port=args.db_port,
        dbname=args.db_name,
        user=args.db_user,
        password=args.db_password,
    )
    try:
        backdate(conn, args.days)
    finally:
        conn.close()


if __name__ == "__main__":
    main()

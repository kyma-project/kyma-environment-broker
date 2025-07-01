BEGIN;

DO $$ BEGIN
    CREATE TYPE action_type AS ENUM ('plan_update', 'subaccount_movement');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

CREATE TABLE IF NOT EXISTS actions (
    id                      varchar(255) NOT NULL PRIMARY KEY,
    type                    action_type NOT NULL,
    instance_id             varchar(255) references instances (instance_id) ON DELETE SET NULL,
    instance_archived_id    varchar(255) references instances_archived (instance_id),
    message                 text NOT NULL,
    old_value               varchar(255) NOT NULL,
    new_value               varchar(255) NOT NULL,
    created_at              timestamp with time zone NOT NULL
);

CREATE INDEX IF NOT EXISTS actions_instance_id ON actions USING HASH (instance_id);
CREATE INDEX IF NOT EXISTS actions_instance_archived_id ON actions USING HASH (instance_archived_id);

COMMIT;

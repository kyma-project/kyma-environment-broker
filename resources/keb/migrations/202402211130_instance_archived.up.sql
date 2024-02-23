BEGIN;

CREATE TABLE IF NOT EXISTS instances_archived (
    instance_id                      varchar(64) NOT NULL PRIMARY KEY,
    global_account_id                varchar(64) NOT NULL,
    last_runtime_id                  varchar(64) NOT NULL,
    subscription_global_account_id   varchar(64) NOT NULL,
    subaccount_id                    varchar(64) NOT NULL,
    plan_id                          varchar(40) NOT NULL,
    plan_name                        varchar(32) NOT NULL,
    region                           varchar(32) NOT NULL,
    subaccount_region                varchar(32) NOT NULL,
    provider                         varchar(32) NOT NULL,
    shoot_name                       varchar(32) NOT NULL,
    internal_user                    boolean NOT NULL,

    provisioning_started_at           timestamp with time zone NOT NULL,
    provisioning_finished_at          timestamp with time zone NOT NULL,
    first_deprovisioning_started_at   timestamp with time zone NOT NULL,
    first_deprovisioning_finished_at  timestamp with time zone NOT NULL,
    last_deprovisioning_finished_at   timestamp with time zone NOT NULL

);


COMMIT;
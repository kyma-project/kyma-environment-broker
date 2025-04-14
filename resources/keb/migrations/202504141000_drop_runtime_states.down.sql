CREATE TABLE IF NOT EXISTS runtime_states
(
    id             character varying(255)   NOT NULL,
    runtime_id     character varying(255),
    operation_id   character varying(255),
    created_at     timestamp with time zone NOT NULL,
    kyma_config    text,
    cluster_config text,
    k8s_version    text,
    cluster_setup  text DEFAULT ''::text
);

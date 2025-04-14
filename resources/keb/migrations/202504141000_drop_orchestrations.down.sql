CREATE TABLE IF NOT EXISTS orchestrations
(
    orchestration_id   character varying(255)                                         NOT NULL,
    created_at         timestamp with time zone                                       NOT NULL,
    updated_at         timestamp with time zone                                       NOT NULL,
    state              character varying(32)                                          NOT NULL,
    parameters         text                                                           NOT NULL,
    description        text,
    runtime_operations text,
    type               character varying(32) DEFAULT 'upgradeKyma'::character varying NOT NULL
);

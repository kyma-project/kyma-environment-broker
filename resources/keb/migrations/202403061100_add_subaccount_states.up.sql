CREATE TABLE IF NOT EXISTS subaccount_states (
    id varchar(255) PRIMARY KEY,
    beta_enabled VARCHAR(255) NOT NULL,
    used_for_production VARCHAR(255),
    modified_at BIGINT NOT NULL
);

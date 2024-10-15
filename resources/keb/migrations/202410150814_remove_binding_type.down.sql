ALTER TABLE bindings
    ADD COLUMN binding_type VARCHAR(64) NOT NULL DEFAULT 'service_account';


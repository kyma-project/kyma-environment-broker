ALTER TABLE instances
    DROP COLUMN encryption_mode;
ALTER TABLE operations
    DROP COLUMN encryption_mode;
ALTER TABLE secret_bindings
    DROP COLUMN encryption_mode;


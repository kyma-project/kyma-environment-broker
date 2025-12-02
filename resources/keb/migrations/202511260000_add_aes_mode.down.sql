ALTER TABLE instances
    DROP COLUMN encryption_mode;
ALTER TABLE operations
    DROP COLUMN encryption_mode;
ALTER TABLE bindings
    DROP COLUMN encryption_mode;


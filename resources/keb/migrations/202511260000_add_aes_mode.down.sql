ALTER TABLE instances
    DROP COLUMN encryption_mode;
ALTER TABLE operations
    DROP COLUMN encryption_mode;
ALTER TABLE bindings
    DROP COLUMN encryption_mode;

DROP INDEX operations_by_encryption_mode;
DROP INDEX instances_by_encryption_mode;
DROP INDEX bindings_by_encryption_mode;

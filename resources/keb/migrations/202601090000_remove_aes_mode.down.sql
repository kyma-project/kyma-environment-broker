ALTER TABLE instances
    ADD COLUMN encryption_mode varchar(32) DEFAULT 'aes-cfb';
ALTER TABLE operations
    ADD COLUMN encryption_mode varchar(32) DEFAULT 'aes-cfb';
ALTER TABLE bindings
    ADD COLUMN encryption_mode varchar(32) DEFAULT 'aes-cfb';

CREATE INDEX operations_by_encryption_mode ON operations (encryption_mode);
CREATE INDEX instances_by_encryption_mode ON instances (encryption_mode);
CREATE INDEX bindings_by_encryption_mode ON bindings (encryption_mode);
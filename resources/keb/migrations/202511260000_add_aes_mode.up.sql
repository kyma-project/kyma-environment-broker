ALTER TABLE instances
    ADD COLUMN encryption_mode varchar(32) DEFAULT 'aes-cfb';
ALTER TABLE operations
    ADD COLUMN encryption_mode varchar(32) DEFAULT 'aes-cfb';
ALTER TABLE bindings
    ADD COLUMN encryption_mode varchar(32) DEFAULT 'aes-cfb';

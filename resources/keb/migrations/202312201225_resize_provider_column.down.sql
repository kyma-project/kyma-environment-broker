ALTER TABLE instances
    ALTER COLUMN provider TYPE varchar(16) USING substring(provider, 1, 16);

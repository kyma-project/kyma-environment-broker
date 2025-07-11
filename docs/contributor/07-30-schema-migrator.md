# Schema Migrator

Schema Migrator is responsible for Kyma Environment Broker's database schema migrations.

## Development

To modify the database schema, you must add migration files to the `/resources/keb/migrations` directory. Use the [`create_migration` script](/scripts/schemamigrator/create_migration.sh) to generate migration templates. See the [Migrations](https://github.com/golang-migrate/migrate/blob/master/MIGRATIONS.md) document for more details. New migration files are mounted as a [Volume](/resources/keb/templates/migrator-job.yaml#L110) from a [ConfigMap](/resources/keb/templates/keb-migrations.yaml).

Make sure to validate the migration files by running the [validation script](/scripts/schemamigrator/validate.sh).

## Configuration

Use the following environment variables to configure the application:

| Environment Variable | Current Value | Description |
|---------------------|------------------------------|---------------------------------------------------------------|
| **DATABASE_EMBEDDED** | <code>true</code> | - |
| **DB_HOST** | None | - |
| **DB_NAME** | None | - |
| **DB_PASSWORD** | None | - |
| **DB_PORT** | None | - |
| **DB_SSL** | None | - |
| **DB_SSLROOTCERT** | None | - |
| **DB_USER** | None | - |
| **DIRECTION** | None | - |

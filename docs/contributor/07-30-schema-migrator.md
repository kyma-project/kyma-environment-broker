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
| **DB_HOST** | None | Specifies the host of the database. |
| **DB_NAME** | None | Specifies the name of the database. |
| **DB_PASSWORD** | None | Specifies the user password for the database. |
| **DB_PORT** | None | Specifies the port for the database. |
| **DB_SSL** | None | Activates the SSL mode for PostgreSQL. |
| **DB_SSLROOTCERT** | <code>/secrets/cloudsql-sslrootcert/server-ca.pem</code> | Path to the Cloud SQL SSL root certificate file. |
| **DB_TIMEZONE** | None | Specifies the "timezone" parameter in the DB connection URL |
| **DB_USER** | None | Specifies the username for the database. |
| **DIRECTION** | <code>up</code> | Defines the direction of the schema migration, either "up" or "down". |

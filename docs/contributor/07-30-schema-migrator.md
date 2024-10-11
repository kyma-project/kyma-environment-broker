# Schema Migrator

Schema Migrator is responsible for Kyma Environment Broker's database schema migrations.

## Development

To modify the database schema, you need to add migration files to `/resources/keb/migrations` directory. Use the [script](/scripts/schemamigrator/create_migration.sh) to generate migration templates. See [Migrations](https://github.com/golang-migrate/migrate/blob/master/MIGRATIONS.md) document for migration details. New migration files are mounted as a [Volume](/resources/keb/templates/migrator-job.yaml#L110) from a [ConfigMap](/resources/keb/templates/keb-migrations.yaml). 

Make sure to validate the migration files by running the [validation script](/scripts/schemamigrator/validate.sh).

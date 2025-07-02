# Archiver Job

The archiver job is a tool for archiving and cleaning the data about already deprovisioned instances. The archiver job is run once. All data about deprovisioned instances in the future will be archived and cleaned by proper deprovisioning steps.

## Running Modes

### Dry Run

The dry run mode does not perform any changes on the database.

### Deletion

The **APP_PERFORM_DELETION** environment variable specifies whether to perform the deletion of the operations and runtime states from the database.
If the value is set to `false`, the archiver job only archives the data.

## Configuration

| Environment variable | Description                                                                                       | Default value                            |
|---|---------------------------------------------------------------------------------------------------|------------------------------------------|
| **APP_DRY_RUN** | Specifies whether to run the job in the `dry-run` mode.                                           | `true`                                   |
| **APP_LOG_LEVEL** | Specifies the log level for the application. Possible values: `debug`, `info`, `warn`, `error`    | `info`.                                   |
| **APP_BATCH_SIZE** | Specifies the number of instances to be archived in one batch.                                    | `100`                                    |
| **APP_PERFORM_DELETION** | Specifies whether to perform the deletion of the operations and runtime states from the database. | `false`                                  |
| **APP_DATABASE_USER** | Specifies the username for the database.                                                          | `postgres`                               |
| **APP_DATABASE_PASSWORD** | Specifies the user password for the database.                                                     | `password`                               |
| **APP_DATABASE_HOST** | Specifies the host of the database.                                                               | `localhost`                              |
| **APP_DATABASE_PORT** | Specifies the port for the database.                                                              | `5432`                                   |
| **APP_DATABASE_NAME** | Specifies the name of the database.                                                               | `broker`                            |

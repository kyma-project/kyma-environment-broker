# Archiver Job

The archiver job is a tool for archiving and cleaning the data about already deprovisioned instances. The archiver job is run once. All data about deprovisioned instances in the future will be archived and cleaned by proper deprovisioning steps.

## Running Modes

### Dry Run

The dry run mode does not perform any changes on the database.

### Deletion

The **APP_PERFORM_DELETION** environment variable specifies whether to perform the deletion of the operations and runtime states from the database.
If the value is set to `false`, the archiver job only archives the data.

## Configuration

Use the following environment variables to configure the application:

| Environment Variable | Current Value | Description |
|---------------------|------------------------------|---------------------------------------------------------------|
| **APP_LOG_LEVEL** | None | - |
| **APP_BATCH_SIZE** | None | - |
| **APP_DRY_RUN** | None | - |
| **APP_PERFORM_DELETION** | None | - |
| **APP_DATABASE_SECRET_&#x200b;KEY** | None | - |
| **APP_DATABASE_USER** | None | - |
| **APP_DATABASE_&#x200b;PASSWORD** | None | - |
| **APP_DATABASE_HOST** | None | - |
| **APP_DATABASE_PORT** | None | - |
| **APP_DATABASE_NAME** | None | - |
| **APP_DATABASE_SSLMODE** | None | - |
| **APP_DATABASE_&#x200b;SSLROOTCERT** | None | - |

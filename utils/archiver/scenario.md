# Running Archiver scenario

## Gather DB Statistics

Run all the SQL statements defined in the [DB statistics which should be run before the archiver is started](db_scripts.md#db-statistics-which-should-be-run-before-the-archiver-is-started) section.

## Run the Archiver Job with Disabled Deletion

Set the following environment variables:
- **APP_PERFORM_DELETION** set to `false`
- **APP_DRY_RUN**  set to `false`

Run the archiver (`./apply.sh`).

## Verify the Archiver Job's Work

Run all the SQL statements defined in the [Statements to verify the archiver work](db_scripts.md#statements-to-verify-the-archiver-work) section.

## Delete All Archived Instances

Make sure that the number of operations were is not decreased (deletion was not performed).
Delete all rows from the instances_archived table (it will be recreated during next archiver execution).
```sql
delete from instances_archived;
```

## Enable Archiving and Deletion of Operations and Instances at the End of Deprovisioning

1. Set the following configurations for KEB:
```
archiving:
    enabled: true
    dryRun: false
cleaning:
    enabled: true
    dryRun: false
```

2. Wait for KEB to restart with the new configuration.

## Run the Archiver Job with Enabled Deletion

Set the following environment variables:
- **APP_PERFORM_DELETION** set to `true`.
- **APP_DRY_RUN**  set to `false`.

## Verify the Archiver Job's Work

Run all the SQL statements defined in the [Statements to verify the archiver work](db_scripts.md#statements-to-verify-the-archiver-work) section.
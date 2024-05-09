# SQL Statements to Verify the Archiver Job's Work and Check DB for Deprovisioned Instances

## DB Statistics to Be Run Before Starting the Archiver Job

Number of all operations:
```sql
select count(*) from operations;
```

Number of operations that the archiver job will delete (belongs to deprovisioned instances):
```sql
select count(*) from operations where instance_id not in (select instance_id from instances);
```

Number of runtime states:
```sql
select count(*) from runtime_states;
```

## Statements to Verify the Archiver Job's Work
Number of all operations:
```sql
select count(*) from operations;
```

Number of operations that the archiver job will delete (belonging to deprovisioned instances):
```sql
select count(*) from operations where instance_id not in (select instance_id from instances);
```

Number of runtime states:
```sql
select count(*) from runtime_states;
```

Number of archived instances:
```sql
select count(*) from instances_archived;
```
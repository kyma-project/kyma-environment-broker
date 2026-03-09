# Cleaning and Archiving

## Archiving

Kyma Environment Broker (KEB) provides an archiving mechanism to store data about deprovisioned instances.
The archiving mechanism is run at the end of the deprovisioning process (but before cleaning) and stores some data about a deprovisioned instance in the archive table.
Such archived instances can be used for investigations using KCP CLI.

## Cleaning

All data about deprovisioned instances is stored in the database. To keep the database clean and not store any sensitive data, KEB provides a cleanup mechanism.
This mechanism is run at the end of the deprovisioning process and removes all data about a deprovisioned instance from the database. It removes the instance from the database along with all related data, such as the instance's operations and runtime states associated with those operations.

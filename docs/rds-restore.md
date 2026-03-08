# Restoring an RDS Instance

Restoring a database instance from a snapshot creates a new database instance.

## Restore Process
1. **Identify the Snapshot**: Locate the snapshot you want to restore from.
2. **Launch Restore**: Use the restore action to initiate the process.
3. **Specify New Instance Details**: Provide a new DB instance identifier.
4. **Configure Settings**: Review the instance class and storage settings.
5. **Finalize**: The system will create a new instance with the data from the snapshot.

> [!NOTE]
> You cannot restore a snapshot to an existing database instance.

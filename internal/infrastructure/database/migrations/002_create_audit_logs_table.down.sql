-- Drop indexes
DROP INDEX IF EXISTS idx_audit_logs_actor_id;
DROP INDEX IF EXISTS idx_audit_logs_timestamp;
DROP INDEX IF EXISTS idx_audit_logs_instance_id;

-- Drop table
DROP TABLE IF EXISTS audit_logs;

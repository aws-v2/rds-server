-- Drop indexes
DROP INDEX IF EXISTS idx_instances_created_at;
DROP INDEX IF EXISTS idx_instances_status;
DROP INDEX IF EXISTS idx_instances_owner_id;

-- Drop table
DROP TABLE IF EXISTS db_instances;

-- Drop indexes
DROP INDEX IF EXISTS idx_config_history_changed_at;
DROP INDEX IF EXISTS idx_config_history_instance_id;
DROP INDEX IF EXISTS idx_configurations_instance_id;

-- Drop tables
DROP TABLE IF EXISTS configuration_history;
DROP TABLE IF EXISTS configurations;

-- Create configurations table
CREATE TABLE IF NOT EXISTS configurations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id UUID REFERENCES db_instances(id) ON DELETE CASCADE,
    parameter VARCHAR(255) NOT NULL,
    value TEXT NOT NULL,
    applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    applied_by VARCHAR(255) NOT NULL,
    UNIQUE(instance_id, parameter)
);

-- Create configuration history table
CREATE TABLE IF NOT EXISTS configuration_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id UUID REFERENCES db_instances(id) ON DELETE CASCADE,
    parameter VARCHAR(255) NOT NULL,
    old_value TEXT,
    new_value TEXT NOT NULL,
    changed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    changed_by VARCHAR(255) NOT NULL
);

-- Create indexes for faster queries
CREATE INDEX idx_configurations_instance_id ON configurations(instance_id);
CREATE INDEX idx_config_history_instance_id ON configuration_history(instance_id);
CREATE INDEX idx_config_history_changed_at ON configuration_history(changed_at DESC);

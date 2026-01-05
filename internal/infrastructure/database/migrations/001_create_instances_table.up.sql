-- Create database instances table
CREATE TABLE IF NOT EXISTS db_instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    engine VARCHAR(100) NOT NULL,
    port INTEGER NOT NULL,
    host VARCHAR(255) NOT NULL DEFAULT 'localhost',
    db_user VARCHAR(255) NOT NULL,
    db_password TEXT NOT NULL,
    owner_id VARCHAR(255) NOT NULL,
    container_id VARCHAR(255) UNIQUE NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'running',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for faster queries
CREATE INDEX idx_instances_owner_id ON db_instances(owner_id);
CREATE INDEX idx_instances_status ON db_instances(status);
CREATE INDEX idx_instances_created_at ON db_instances(created_at DESC);

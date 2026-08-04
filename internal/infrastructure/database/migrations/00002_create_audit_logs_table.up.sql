-- Create audit logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id UUID REFERENCES db_instances(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL,
    actor_id VARCHAR(255) NOT NULL,
    details JSONB,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for faster queries
CREATE INDEX idx_audit_logs_instance_id ON audit_logs(instance_id);
CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp DESC);
CREATE INDEX idx_audit_logs_actor_id ON audit_logs(actor_id);





-- {"level":"WARN","ts":"2026-08-04T13:00:36.835+0300",
-- "caller":"api/main.go:164",
-- "msg":"Failed to connect to PostgreSQL","attempt":2,
-- "error":"failed to run migrations: migration failed: migration failed: column \"status\" of relation \"databases\" already exists in line 0: CREATE TABLE IF NOT EXISTS vpcs (\n    id VARCHAR(36) PRIMARY KEY,\n    name VARCHAR(255) NOT NULL,\n    cidr_block VARCHAR(50) NOT NULL,\n    bridge_name VARCHAR(100) NOT NULL,\n    subnet VARCHAR(50) NOT NULL,\n    gateway VARCHAR(50) NOT NULL,\n    tenant_id VARCHAR(36) NOT NULL,\n    status VARCHAR(50) NOT NULL,\n    is_default BOOLEAN DEFAULT FALSE,\n    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,\n    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP\n);\n\nCREATE INDEX IF NOT EXISTS idx_vpcs_tenant_id ON vpcs(tenant_id);\n\n\n\n\n\nCREATE TABLE IF NOT EXISTS databases (\n    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),\n    user_id      TEXT NOT NULL,\n    name         TEXT NOT NULL,\n    vm_ip        TEXT NOT NULL,\n    gateway_ip   TEXT NOT NULL,\n    gateway_port INTEGER NOT NULL,\n    vm_db_port   INTEGER NOT NULL DEFAULT 5432,\n    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),\n    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()\n);\n-- Compound unique constraint for the gap-finder to prevent port collisions on the same host\n-- ALTER TABLE databases ADD CONSTRAINT unique_node_host_port UNIQUE (node_host, node_port);\nAlter table databases add column status text not null default '';\nAlter table databases add column deleted_at TIMESTAMP;\n-- Indexes for quick routing and tenant lookups\nCREATE INDEX idx_databases_account_id ON databases(user_id);\n\n (details: pq: column \"status\" of relation \"databases\" already exists)"}
 
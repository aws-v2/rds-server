CREATE TABLE IF NOT EXISTS vpcs (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    cidr_block VARCHAR(50) NOT NULL,
    bridge_name VARCHAR(100) NOT NULL,
    subnet VARCHAR(50) NOT NULL,
    gateway VARCHAR(50) NOT NULL,
    tenant_id VARCHAR(36) NOT NULL,
    status VARCHAR(50) NOT NULL,
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_vpcs_tenant_id ON vpcs(tenant_id);





CREATE TABLE IF NOT EXISTS databases (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      TEXT NOT NULL,
    name         TEXT NOT NULL,
    vm_ip        TEXT NOT NULL,
    gateway_ip   TEXT NOT NULL,
    gateway_port INTEGER NOT NULL,
    vm_db_port   INTEGER NOT NULL DEFAULT 5432,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- Compound unique constraint for the gap-finder to prevent port collisions on the same host
-- ALTER TABLE databases ADD CONSTRAINT unique_node_host_port UNIQUE (node_host, node_port);
Alter table databases add column IF NOT EXISTS status text not null default '';
Alter table databases add column IF NOT EXISTS deleted_at TIMESTAMP;
-- Indexes for quick routing and tenant lookups
CREATE INDEX idx_databases_account_id ON databases(user_id);


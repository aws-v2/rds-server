CREATE TYPE db_status AS ENUM ('PENDING', 'PROVISIONING', 'AVAILABLE', 'MAINTENANCE', 'DELETED', 'FAILED');

CREATE TABLE databases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL,               -- Links to your SaaS User/Tenant ID
    
    -- Identification
    name VARCHAR(255) NOT NULL,             -- User-friendly name (e.g., "Prod API DB")
    physical_db_name VARCHAR(64) UNIQUE,    -- Secure, generated name on the Postgres server (e.g., "db_a7f9b2")
    
    -- Routing
    node_host VARCHAR(255) NOT NULL,        -- The EC2 instance IP or Internal DNS
    node_port INTEGER DEFAULT 5432,         -- Port of the Postgres container
    
    -- Status & Lifecycle
    status db_status DEFAULT 'PENDING',
    
    -- Idempotency
    idempotency_key UUID UNIQUE,            -- Optional idempotency key for creation requests

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ                  -- Soft deletes for recovery purposes
);

-- Compound unique constraint for the gap-finder to prevent port collisions on the same host
ALTER TABLE databases ADD CONSTRAINT unique_node_host_port UNIQUE (node_host, node_port);

-- Indexes for quick routing and tenant lookups
CREATE INDEX idx_databases_account_id ON databases(account_id);


CREATE TYPE credential_status AS ENUM ('ACTIVE', 'ROTATING', 'REVOKED');

CREATE TABLE credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    database_id UUID REFERENCES databases(id) ON DELETE CASCADE,
    
    -- Postgres Role 
    role_name VARCHAR(64) UNIQUE NOT NULL,      -- e.g., "role_a7f9b2_master"
    encrypted_password TEXT,                    -- Symmetric encrypted password (KMS)
    
    is_master BOOLEAN DEFAULT FALSE,            -- Does this role own the DB?
    status credential_status DEFAULT 'ACTIVE',
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);


CREATE TYPE backup_type AS ENUM ('AUTOMATED', 'MANUAL');
CREATE TYPE backup_status AS ENUM ('PENDING', 'RUNNING', 'COMPLETED', 'FAILED', 'DELETED');

CREATE TABLE backups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    database_id UUID REFERENCES databases(id) ON DELETE CASCADE,
    
    type backup_type NOT NULL,
    status backup_status DEFAULT 'PENDING',
    
    -- Storage
    storage_uri TEXT,                           -- e.g., "s3://claudedb-backups/db_a7f9b2/1678888.sql.gz"
    size_bytes BIGINT,
    
    -- Lifecycle
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,                     -- For automatic cleanup of old automated backups
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_backups_database_id ON backups(database_id);
CREATE INDEX idx_backups_expires_at ON backups(expires_at) WHERE status = 'COMPLETED';


CREATE TYPE operation_type AS ENUM ('PROVISION_DB', 'DELETE_DB', 'CREATE_BACKUP', 'RESTORE_BACKUP', 'ROTATE_CREDS');
CREATE TYPE operation_status AS ENUM ('QUEUED', 'RUNNING', 'SUCCESS', 'FAILED');

CREATE TABLE operations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    database_id UUID REFERENCES databases(id) ON DELETE CASCADE,
    
    type operation_type NOT NULL,
    status operation_status DEFAULT 'QUEUED',
    
    error_message TEXT,                         -- Populated if the worker fails
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

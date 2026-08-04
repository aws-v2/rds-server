CREATE TABLE volumes (
    id VARCHAR(36) PRIMARY KEY,
    account_id VARCHAR(36) NOT NULL,
    arn VARCHAR(255),
    name VARCHAR(255) NOT NULL,
    size_gb INT NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP
);

CREATE TABLE snapshots (
    id VARCHAR(36) PRIMARY KEY,
    account_id VARCHAR(36) NOT NULL,
    arn VARCHAR(255),
    name VARCHAR(255) NOT NULL,
    database_id VARCHAR(36) NOT NULL,
    volume_id VARCHAR(36) NOT NULL,
    size_gb INT NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP
);

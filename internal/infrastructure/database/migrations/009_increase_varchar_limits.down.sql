-- We do not strictly need to down-migrate VARCHAR(255) to VARCHAR(36) 
-- as it could cause truncation errors. 
-- However, for completeness:

-- ALTER TABLE volumes ALTER COLUMN id TYPE VARCHAR(36);
-- ALTER TABLE volumes ALTER COLUMN account_id TYPE VARCHAR(36);
-- ALTER TABLE snapshots ALTER COLUMN id TYPE VARCHAR(36);
-- ALTER TABLE snapshots ALTER COLUMN account_id TYPE VARCHAR(36);
-- ALTER TABLE snapshots ALTER COLUMN database_id TYPE VARCHAR(36);
-- ALTER TABLE snapshots ALTER COLUMN volume_id TYPE VARCHAR(36);

SELECT 1;

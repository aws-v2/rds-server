-- Drop the existing global unique constraint on role_name
ALTER TABLE credentials DROP CONSTRAINT IF EXISTS credentials_role_name_key;

-- Add a new unique constraint that ensures role_name is only unique per database
ALTER TABLE credentials ADD CONSTRAINT credentials_database_id_role_name_key UNIQUE (database_id, role_name);

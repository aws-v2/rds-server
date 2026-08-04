-- Revert the constraint back to global unique (this may fail if there are duplicates)
ALTER TABLE credentials DROP CONSTRAINT IF EXISTS credentials_database_id_role_name_key;
ALTER TABLE credentials ADD CONSTRAINT credentials_role_name_key UNIQUE (role_name);

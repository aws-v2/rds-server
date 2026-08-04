-- Removing enum values in Postgres is complex and often requires recreating the type 
-- or deleting the rows referencing them first. We will use a safe stub here.
-- Down migrations for adding enum values are generally left empty or require full table rewrites.
SELECT 1;

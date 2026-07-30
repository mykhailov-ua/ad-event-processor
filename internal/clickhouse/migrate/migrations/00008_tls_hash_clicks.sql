ALTER TABLE clicks ADD COLUMN IF NOT EXISTS tls_hash String DEFAULT '' AFTER user_agent;

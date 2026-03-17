-- runs: always

DO $$
BEGIN
  -- Phase 1: Add new columns if they don't exist yet
  ALTER TABLE config_access_logs
    ADD COLUMN IF NOT EXISTS id TEXT NOT NULL DEFAULT generate_ulid(),
    ADD COLUMN IF NOT EXISTS external_role_id UUID,
    ADD COLUMN IF NOT EXISTS client_ip TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS verb TEXT,
    ADD COLUMN IF NOT EXISTS outcome TEXT NOT NULL DEFAULT 'allowed',
    ADD COLUMN IF NOT EXISTS mfa BOOLEAN,
    ADD COLUMN IF NOT EXISTS fingerprint TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS first_observed TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS inserted_at TIMESTAMPTZ DEFAULT now(),
    ADD COLUMN IF NOT EXISTS bucket_start TIMESTAMPTZ DEFAULT date_trunc('day', now());

  -- Phase 2: Backfill existing rows
  UPDATE config_access_logs SET
    client_ip = COALESCE(properties->>'ip_address', ''),
    outcome = 'allowed',
    first_observed = created_at,
    bucket_start = date_trunc('day', created_at),
    fingerprint = md5(concat_ws('|',
      config_id::text, scraper_id::text, '',
      external_user_id::text, 'allowed',
      COALESCE(mfa::text, 'false'), date_trunc('day', created_at)::text
    ))
  WHERE first_observed IS NULL;

  -- Phase 3: Seed N/A external role for use as NOT NULL default
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'external_roles') THEN
    IF NOT EXISTS (SELECT 1 FROM external_roles WHERE id = '00000000-0000-0000-0000-000000000000') THEN
      INSERT INTO external_roles (id, account_id, scraper_id, role_type, name, description)
      VALUES ('00000000-0000-0000-0000-000000000000', 'system', '00000000-0000-0000-0000-000000000000', 'system', 'N/A', 'Sentinel role for access logs without a specific role');
    END IF;
  END IF;

  -- Phase 4: Backfill NOT NULL columns before Atlas enforces constraints
  UPDATE config_access_logs SET external_role_id = '00000000-0000-0000-0000-000000000000' WHERE external_role_id IS NULL;
  UPDATE config_access_logs SET mfa = false WHERE mfa IS NULL;
  UPDATE config_access_logs SET bucket_start = date_trunc('day', created_at) WHERE bucket_start IS NULL;

  -- Phase 5: Drop old composite PK if it targets config_id (idempotent)
  IF EXISTS (
    SELECT 1 FROM pg_constraint c
    JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = ANY(c.conkey)
    WHERE c.conname = 'config_access_logs_pkey'
      AND c.conrelid = 'config_access_logs'::regclass
      AND a.attname = 'config_id'
  ) THEN
    ALTER TABLE config_access_logs DROP CONSTRAINT config_access_logs_pkey;
    ALTER TABLE config_access_logs ADD PRIMARY KEY (id);
  END IF;
END $$;

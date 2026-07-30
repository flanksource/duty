-- Durable external-user identity mappings. The external_users.aliases array is
-- retained for compatibility, while this table provides a normalized lookup
-- for scraped aliases, manually-added aliases, and historical IDs after merge.

CREATE OR REPLACE FUNCTION normalize_external_user_alias()
RETURNS TRIGGER AS $$
BEGIN
  NEW.alias := lower(btrim(NEW.alias));
  IF NEW.alias = '' THEN
    RAISE EXCEPTION 'external user alias cannot be empty' USING ERRCODE = '23514';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS normalize_external_user_alias_trigger ON external_user_aliases;
CREATE TRIGGER normalize_external_user_alias_trigger
  BEFORE INSERT OR UPDATE OF alias ON external_user_aliases
  FOR EACH ROW
  EXECUTE FUNCTION normalize_external_user_alias();

-- Keep scraper-discovered aliases represented in the mapping table. Canonical
-- IDs are resolved directly from external_users; only historical loser IDs are
-- stored as aliases after a merge. Conflicts are intentionally left with their
-- existing owner so normal scraper resolution can reconcile duplicate users.
CREATE OR REPLACE FUNCTION sync_external_user_aliases()
RETURNS TRIGGER AS $$
BEGIN
  IF NEW.deleted_at IS NOT NULL THEN
    RETURN NULL;
  END IF;

  INSERT INTO external_user_aliases (external_user_id, alias, source)
  SELECT NEW.id, lower(btrim(a)), 'scrape'
  FROM unnest(COALESCE(NEW.aliases, '{}'::text[])) AS a
  WHERE btrim(a) <> ''
  ON CONFLICT (alias) WHERE deleted_at IS NULL DO NOTHING;

  RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS sync_external_user_aliases_trigger ON external_users;
CREATE TRIGGER sync_external_user_aliases_trigger
  AFTER INSERT OR UPDATE OF aliases, deleted_at ON external_users
  FOR EACH ROW
  EXECUTE FUNCTION sync_external_user_aliases();

-- Backfill existing active users. If historical data contains an overlapping
-- alias, choose the same deterministic lowest UUID used by the scrape merge
-- function; a subsequent merge will consolidate the duplicate rows.
INSERT INTO external_user_aliases (external_user_id, alias, source)
SELECT DISTINCT ON (key.alias) key.external_user_id, key.alias, 'scrape'
FROM (
  SELECT eu.id AS external_user_id, lower(btrim(a)) AS alias
  FROM external_users eu
  CROSS JOIN LATERAL unnest(COALESCE(eu.aliases, '{}'::text[])) AS a
  WHERE eu.deleted_at IS NULL AND btrim(a) <> ''
) key
ORDER BY key.alias, key.external_user_id::text
ON CONFLICT (alias) WHERE deleted_at IS NULL DO NOTHING;

-- Add a non-conflicting manual alias. Aliases which already identify another
-- user require merge_external_users so adding a typo cannot silently merge
-- identities and rewrite access records.
CREATE OR REPLACE FUNCTION add_external_user_alias(
  p_external_user_id UUID,
  p_alias TEXT,
  p_created_by UUID DEFAULT NULL
)
RETURNS external_user_aliases AS $$
DECLARE
  v_alias TEXT := lower(btrim(p_alias));
  v_existing external_user_aliases%ROWTYPE;
  v_result external_user_aliases%ROWTYPE;
  v_owner UUID;
BEGIN
  IF v_alias = '' THEN
    RAISE EXCEPTION 'external user alias cannot be empty' USING ERRCODE = '23514';
  END IF;

  IF NOT EXISTS (
    SELECT 1 FROM external_users
    WHERE id = p_external_user_id AND deleted_at IS NULL
  ) THEN
    RAISE EXCEPTION 'active external user % does not exist', p_external_user_id
      USING ERRCODE = '23503';
  END IF;

  SELECT * INTO v_existing
  FROM external_user_aliases
  WHERE alias = v_alias AND deleted_at IS NULL
  FOR UPDATE;

  IF FOUND THEN
    IF v_existing.external_user_id <> p_external_user_id THEN
      RAISE EXCEPTION 'alias % is already assigned to external user %', v_alias, v_existing.external_user_id
        USING ERRCODE = '23505';
    END IF;
    RETURN v_existing;
  END IF;

  -- Live canonical UUIDs are intentionally not duplicated in the mapping
  -- table, so check external_users as well before accepting a manual alias.
  SELECT id INTO v_owner
  FROM external_users
  WHERE deleted_at IS NULL
    AND id <> p_external_user_id
    AND (id::text = v_alias OR v_alias = ANY(COALESCE(aliases, '{}'::text[])))
  LIMIT 1;
  IF FOUND THEN
    RAISE EXCEPTION 'alias % already identifies active external user %', v_alias, v_owner
      USING ERRCODE = '23505';
  END IF;

  INSERT INTO external_user_aliases (external_user_id, alias, source, created_by)
  VALUES (p_external_user_id, v_alias, 'manual', p_created_by)
  RETURNING * INTO v_result;

  -- Canonical IDs live in the mapping table but are not duplicated in the
  -- compatibility aliases array.
  IF v_alias <> p_external_user_id::text THEN
    UPDATE external_users
    SET aliases = ARRAY(
      SELECT DISTINCT a
      FROM unnest(COALESCE(aliases, '{}'::text[]) || ARRAY[v_alias]) AS a
      ORDER BY a
    ), updated_at = now()
    WHERE id = p_external_user_id AND deleted_at IS NULL;
  END IF;

  RETURN v_result;
END;
$$ LANGUAGE plpgsql;

-- Merge a duplicate user into an explicitly-selected primary user. Unlike
-- merge_and_upsert_external_users, this function never chooses a winner by UUID
-- ordering: p_primary_id always survives.
CREATE OR REPLACE FUNCTION merge_external_users(
  p_primary_id UUID,
  p_duplicate_id UUID,
  p_created_by UUID DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
  v_primary external_users%ROWTYPE;
  v_duplicate external_users%ROWTYPE;
  v_aliases TEXT[];
  v_conflict external_user_aliases%ROWTYPE;
  v_redirect UUID;
BEGIN
  IF p_primary_id = p_duplicate_id THEN
    IF EXISTS (SELECT 1 FROM external_users WHERE id = p_primary_id AND deleted_at IS NULL) THEN
      RETURN p_primary_id;
    END IF;
    RAISE EXCEPTION 'active external user % does not exist', p_primary_id USING ERRCODE = '23503';
  END IF;

  LOCK TABLE config_access, access_reviews, config_access_logs, external_user_groups,
    external_user_aliases, external_users IN SHARE ROW EXCLUSIVE MODE;

  SELECT * INTO v_primary
  FROM external_users
  WHERE id = p_primary_id AND deleted_at IS NULL
  FOR UPDATE;
  IF NOT FOUND THEN
    RAISE EXCEPTION 'active primary external user % does not exist', p_primary_id USING ERRCODE = '23503';
  END IF;

  SELECT * INTO v_duplicate
  FROM external_users
  WHERE id = p_duplicate_id
  FOR UPDATE;
  IF NOT FOUND THEN
    RAISE EXCEPTION 'duplicate external user % does not exist', p_duplicate_id USING ERRCODE = '23503';
  END IF;

  -- Make retries idempotent after a successful merge.
  IF v_duplicate.deleted_at IS NOT NULL THEN
    SELECT external_user_id INTO v_redirect
    FROM external_user_aliases
    WHERE alias = p_duplicate_id::text AND deleted_at IS NULL;
    IF v_redirect = p_primary_id THEN
      RETURN p_primary_id;
    END IF;
    RAISE EXCEPTION 'duplicate external user % is already deleted', p_duplicate_id USING ERRCODE = '23503';
  END IF;

  SELECT array_agg(DISTINCT normalized ORDER BY normalized) INTO v_aliases
  FROM (
    SELECT lower(btrim(a)) AS normalized
    FROM unnest(
      COALESCE(v_duplicate.aliases, '{}'::text[])
      || ARRAY[v_duplicate.id::text]
      || CASE
           WHEN v_duplicate.email IS NOT NULL AND btrim(v_duplicate.email) <> ''
           THEN ARRAY[lower(btrim(v_duplicate.email))]
           ELSE '{}'::text[]
         END
    ) AS a
    WHERE btrim(a) <> ''
  ) aliases;

  -- An alias owned by a third user indicates a larger connected component.
  -- Do not silently steal it during a two-user manual merge.
  SELECT eua.* INTO v_conflict
  FROM external_user_aliases eua
  WHERE eua.deleted_at IS NULL
    AND eua.alias = ANY(COALESCE(v_aliases, '{}'::text[]))
    AND eua.external_user_id NOT IN (p_primary_id, p_duplicate_id)
  LIMIT 1;
  IF FOUND THEN
    RAISE EXCEPTION 'alias % is assigned to external user %, which is not part of this merge',
      v_conflict.alias, v_conflict.external_user_id
      USING ERRCODE = '23505';
  END IF;

  SELECT id INTO v_redirect
  FROM external_users
  WHERE deleted_at IS NULL
    AND id NOT IN (p_primary_id, p_duplicate_id)
    AND (
      id::text = ANY(COALESCE(v_aliases, '{}'::text[]))
      OR aliases && COALESCE(v_aliases, '{}'::text[])
    )
  LIMIT 1;
  IF FOUND THEN
    RAISE EXCEPTION 'duplicate aliases also identify external user %, which is not part of this merge', v_redirect
      USING ERRCODE = '23505';
  END IF;

  -- Soft-delete active duplicate grants which would collide after remapping.
  WITH candidates AS (
    SELECT ca.id,
           EXISTS (
             SELECT 1
             FROM config_access existing
             WHERE existing.deleted_at IS NULL
               AND existing.id <> ca.id
               AND existing.config_id = ca.config_id
               AND existing.external_user_id = p_primary_id
               AND existing.external_group_id IS NOT DISTINCT FROM ca.external_group_id
               AND existing.external_role_id IS NOT DISTINCT FROM ca.external_role_id
           ) AS collides_with_primary,
           ROW_NUMBER() OVER (
             PARTITION BY ca.config_id, ca.external_group_id, ca.external_role_id
             ORDER BY ca.created_at, ca.id
           ) AS target_rank
    FROM config_access ca
    WHERE ca.external_user_id = p_duplicate_id AND ca.deleted_at IS NULL
  )
  UPDATE config_access ca
  SET deleted_at = now()
  FROM candidates c
  WHERE ca.id = c.id AND (c.collides_with_primary OR c.target_rank > 1);

  UPDATE config_access
  SET external_user_id = p_primary_id
  WHERE external_user_id = p_duplicate_id;

  UPDATE access_reviews
  SET external_user_id = p_primary_id
  WHERE external_user_id = p_duplicate_id;

  INSERT INTO config_access_logs (config_id, external_user_id, scraper_id, created_at, mfa, properties, count)
  SELECT config_id,
         p_primary_id,
         scraper_id,
         MAX(created_at),
         (ARRAY_AGG(mfa ORDER BY created_at DESC))[1],
         (ARRAY_AGG(properties ORDER BY created_at DESC))[1],
         COALESCE(SUM(COALESCE(count, 1)), 0)::integer
  FROM config_access_logs
  WHERE external_user_id = p_duplicate_id
  GROUP BY config_id, scraper_id
  ON CONFLICT (config_id, external_user_id, scraper_id) DO UPDATE SET
    count = COALESCE(config_access_logs.count, 0) + COALESCE(EXCLUDED.count, 0),
    created_at = GREATEST(config_access_logs.created_at, EXCLUDED.created_at),
    mfa = CASE WHEN EXCLUDED.created_at >= config_access_logs.created_at THEN EXCLUDED.mfa ELSE config_access_logs.mfa END,
    properties = CASE WHEN EXCLUDED.created_at >= config_access_logs.created_at THEN EXCLUDED.properties ELSE config_access_logs.properties END;

  DELETE FROM config_access_logs WHERE external_user_id = p_duplicate_id;

  INSERT INTO external_user_groups (external_user_id, external_group_id, scraper_id, created_at)
  SELECT p_primary_id, external_group_id, scraper_id, created_at
  FROM external_user_groups
  WHERE external_user_id = p_duplicate_id AND deleted_at IS NULL
  ON CONFLICT (external_user_id, external_group_id, scraper_id) DO NOTHING;

  DELETE FROM external_user_groups WHERE external_user_id = p_duplicate_id;

  UPDATE external_user_aliases
  SET external_user_id = p_primary_id,
      source = 'merge',
      created_by = COALESCE(created_by, p_created_by)
  WHERE external_user_id = p_duplicate_id AND deleted_at IS NULL;

  INSERT INTO external_user_aliases (external_user_id, alias, source, created_by)
  SELECT p_primary_id, alias, 'merge', p_created_by
  FROM unnest(COALESCE(v_aliases, '{}'::text[])) AS alias
  ON CONFLICT (alias) WHERE deleted_at IS NULL DO NOTHING;

  UPDATE external_users
  SET aliases = NULLIF(ARRAY(
        SELECT DISTINCT a
        FROM unnest(COALESCE(aliases, '{}'::text[]) || COALESCE(v_aliases, '{}'::text[])) AS a
        WHERE a <> ''
        ORDER BY a
      ), '{}'::text[]),
      updated_at = now()
  WHERE id = p_primary_id;

  UPDATE external_users
  SET deleted_at = now(), updated_at = now()
  WHERE id = p_duplicate_id;

  RETURN p_primary_id;
END;
$$ LANGUAGE plpgsql;

-- _debug_log: generic debug helper for instrumenting plpgsql functions.
-- Gated on session setting `debug_log.enabled = 'on'`. Emits the step +
-- detail as a `RAISE NOTICE` so the output survives savepoint rollback
-- (NOTICE messages stream to the client out-of-band and are not rolled
-- back by transaction/savepoint aborts — unlike INSERTs into any table,
-- temp or otherwise).
--
-- The NOTICE payload is prefixed with a sentinel `_DEBUG_LOG:` so callers
-- (both psql and the Go dump helper) can grep for it among other notices.
--
-- Usage:
--   SET LOCAL debug_log.enabled = 'on';
--   ... call the instrumented function inside a savepoint ...
--   -- NOTICE lines appear in the client stream even if the call failed.
CREATE OR REPLACE FUNCTION _debug_log(p_step TEXT, p_detail JSONB) RETURNS void AS $$
BEGIN
  IF current_setting('debug_log.enabled', true) IS DISTINCT FROM 'on' THEN
    RETURN;
  END IF;
  RAISE NOTICE '_DEBUG_LOG: % %', p_step, COALESCE(p_detail::text, 'null');
END;
$$ LANGUAGE plpgsql;


-- merge_and_upsert_external_users: detects alias overlaps within temp table
-- and against live table, remaps FKs, merges aliases, soft-deletes losers, upserts survivors.
-- Returns (loser_id, winner_id) pairs for caller cache eviction.
CREATE OR REPLACE FUNCTION merge_and_upsert_external_users(p_temp_table TEXT)
RETURNS TABLE(loser_id UUID, winner_id UUID) AS $$
DECLARE
  v_debug BOOLEAN := current_setting('debug_log.enabled', true) = 'on';
  v_row_count BIGINT;
  v_edges_total BIGINT;
  v_temp_live_count BIGINT;
  v_temp_live_no_del_filter BIGINT;
  v_temp_live_null_aliases BIGINT;
  v_temp_live_id_eq BIGINT;
  v_sample JSONB;
BEGIN
  LOCK TABLE config_access, access_reviews, config_access_logs, external_user_groups,
    external_users, external_groups, external_roles IN SHARE ROW EXCLUSIVE MODE;

  IF v_debug THEN
    EXECUTE format('SELECT count(*) FROM %I', p_temp_table) INTO v_row_count;
    EXECUTE format('SELECT jsonb_agg(to_jsonb(t)) FROM (SELECT * FROM %I LIMIT 5) t', p_temp_table) INTO v_sample;
    PERFORM _debug_log('entry', jsonb_build_object(
      'temp_table', p_temp_table,
      'row_count', v_row_count,
      'sample', v_sample
    ));
  END IF;

  -- Step 0: Normalize the temp table's aliases to match the normalization
  -- that the `normalize_aliases` BEFORE INSERT/UPDATE trigger applies to
  -- rows stored in external_users (lowercase + sorted + distinct). The
  -- edge-build query below uses `tmp.aliases && live.aliases`, which is
  -- byte-exact, so any mixed-case alias in the temp table would miss
  -- matches against live rows whose aliases the trigger has already
  -- normalized. Normalizing up-front keeps the && join symmetric and
  -- reliable.
  EXECUTE format('
    UPDATE %1$I SET
      aliases = ARRAY(SELECT DISTINCT LOWER(elem) FROM unnest(aliases) AS elem ORDER BY LOWER(elem))
    WHERE aliases IS NOT NULL
  ', p_temp_table);

  IF v_debug THEN
    GET DIAGNOSTICS v_row_count = ROW_COUNT;
    PERFORM _debug_log('step0_normalize_temp_aliases', jsonb_build_object('rows_affected', v_row_count));
  END IF;

  -- Step 1: Build undirected edge list from alias overlaps
  EXECUTE format('
    CREATE TEMP TABLE _eu_edges ON COMMIT DROP AS
    SELECT DISTINCT a.id AS id1, b.id AS id2
    FROM %1$I a JOIN %1$I b ON a.id::text < b.id::text AND a.aliases && b.aliases
    UNION
    SELECT DISTINCT CASE WHEN tmp.id::text < live.id::text THEN tmp.id ELSE live.id END,
                    CASE WHEN tmp.id::text < live.id::text THEN live.id ELSE tmp.id END
    FROM %1$I tmp JOIN external_users live
      ON tmp.aliases && live.aliases AND tmp.id != live.id AND live.deleted_at IS NULL
  ', p_temp_table);

  IF v_debug THEN
    -- Post Step 1: edge counts + sample edges.
    SELECT count(*) INTO v_edges_total FROM _eu_edges;
    PERFORM _debug_log('step1_edges', jsonb_build_object(
      'edge_count', v_edges_total,
      'sample_edges', (SELECT jsonb_agg(jsonb_build_object('id1', id1, 'id2', id2))
                       FROM (SELECT * FROM _eu_edges LIMIT 20) e)
    ));

    -- Diagnostic re-query: same join with variations to pinpoint which filter
    -- is dropping overlaps. Uses unnest + = ANY to bypass the && operator in
    -- case &&'s behavior is the culprit.
    EXECUTE format('
      SELECT count(*) FROM %1$I tmp JOIN external_users live
        ON tmp.aliases && live.aliases AND tmp.id != live.id AND live.deleted_at IS NULL
    ', p_temp_table) INTO v_temp_live_count;
    EXECUTE format('
      SELECT count(*) FROM %1$I tmp JOIN external_users live
        ON tmp.aliases && live.aliases AND tmp.id != live.id
    ', p_temp_table) INTO v_temp_live_no_del_filter;
    EXECUTE format('
      SELECT count(*) FROM %1$I tmp JOIN external_users live
        ON tmp.id = live.id
    ', p_temp_table) INTO v_temp_live_id_eq;
    EXECUTE format('
      SELECT count(*) FROM %1$I tmp
      JOIN LATERAL unnest(COALESCE(tmp.aliases, ARRAY[]::text[])) AS t_alias(val) ON true
      JOIN external_users live ON live.deleted_at IS NULL AND live.id <> tmp.id
      WHERE t_alias.val = ANY(COALESCE(live.aliases, ARRAY[]::text[]))
    ', p_temp_table) INTO v_temp_live_null_aliases;
    PERFORM _debug_log('step1_candidates', jsonb_build_object(
      'temp_live_count_using_overlap_op', v_temp_live_count,
      'temp_live_count_without_deleted_filter', v_temp_live_no_del_filter,
      'temp_live_count_with_id_eq', v_temp_live_id_eq,
      'temp_live_count_via_unnest_any', v_temp_live_null_aliases
    ));
  END IF;

  IF NOT EXISTS (SELECT 1 FROM _eu_edges) THEN
    IF v_debug THEN
      PERFORM _debug_log('short_circuit_entry', jsonb_build_object('reason', 'empty _eu_edges'));

      -- Post-hoc: find any temp↔live pair that overlaps via unnest + = ANY.
      -- If this returns rows, the && operator missed them and the short-
      -- circuit branch will fail with a partial unique index violation.
      EXECUTE format('
        SELECT jsonb_agg(row_to_json(r))
        FROM (
          SELECT tmp.id AS tmp_id, tmp.aliases AS tmp_aliases,
                 live.id AS live_id, live.aliases AS live_aliases,
                 t_alias.val AS shared_alias
          FROM %1$I tmp
          JOIN LATERAL unnest(COALESCE(tmp.aliases, ARRAY[]::text[])) AS t_alias(val) ON true
          JOIN external_users live
            ON live.deleted_at IS NULL
            AND live.id <> tmp.id
            AND t_alias.val = ANY(COALESCE(live.aliases, ARRAY[]::text[]))
          LIMIT 20
        ) r
      ', p_temp_table) INTO v_sample;
      PERFORM _debug_log('short_circuit_collisions_via_unnest', v_sample);
    END IF;

    EXECUTE format('
      INSERT INTO external_users (id, aliases, name, account_id, user_type, email, scraper_id, created_at, updated_at, created_by)
      SELECT id, aliases, name, account_id, user_type, email, scraper_id, created_at, updated_at, created_by FROM %I
      ON CONFLICT (id) DO UPDATE SET
        aliases = NULLIF(ARRAY(SELECT DISTINCT unnest FROM unnest(external_users.aliases || EXCLUDED.aliases) ORDER BY 1), ''{}''::text[]),
        name = EXCLUDED.name, account_id = EXCLUDED.account_id,
        user_type = EXCLUDED.user_type, email = EXCLUDED.email,
        updated_at = EXCLUDED.updated_at, deleted_at = NULL
    ', p_temp_table);
    RETURN;
  END IF;

  -- Step 2: Connected components via label propagation (iterative fixpoint)
  CREATE TEMP TABLE _eu_comp (node UUID PRIMARY KEY, leader UUID) ON COMMIT DROP;
  INSERT INTO _eu_comp (node, leader)
    SELECT DISTINCT id1, id1 FROM _eu_edges UNION SELECT DISTINCT id2, id2 FROM _eu_edges;

  LOOP
    UPDATE _eu_comp c SET leader = sub.new_leader::uuid
    FROM (
      SELECT c2.node, LEAST(c2.leader::text, MIN(e.min_neighbor::text)) AS new_leader
      FROM _eu_comp c2
      JOIN (
        SELECT id1, MIN(c3.leader::text)::uuid AS min_neighbor FROM _eu_edges JOIN _eu_comp c3 ON c3.node = id2 GROUP BY id1
        UNION ALL
        SELECT id2, MIN(c3.leader::text)::uuid AS min_neighbor FROM _eu_edges JOIN _eu_comp c3 ON c3.node = id1 GROUP BY id2
      ) e ON e.id1 = c2.node
      GROUP BY c2.node, c2.leader
      HAVING LEAST(c2.leader::text, MIN(e.min_neighbor::text)) < c2.leader::text
    ) sub
    WHERE c.node = sub.node;

    EXIT WHEN NOT FOUND;
  END LOOP;

  -- Step 3: Build merge pairs (every non-leader -> leader)
  CREATE TEMP TABLE _eu_merges (loser_id UUID PRIMARY KEY, winner_id UUID) ON COMMIT DROP;
  INSERT INTO _eu_merges (loser_id, winner_id)
    SELECT node, leader FROM _eu_comp WHERE node != leader;

  IF v_debug THEN
    PERFORM _debug_log('step3_merges', jsonb_build_object(
      'comp_count', (SELECT count(*) FROM _eu_comp),
      'merges_count', (SELECT count(*) FROM _eu_merges),
      'sample_merges', (SELECT jsonb_agg(jsonb_build_object('loser', m.loser_id, 'winner', m.winner_id))
                        FROM (SELECT * FROM _eu_merges LIMIT 20) m)
    ));
  END IF;

  -- Step 3a: Pre-soft-delete any live losers BEFORE Step 3b inserts the temp
  -- winners. Without this, the partial unique index on aliases still contains
  -- the live loser row at the moment the temp winner is inserted, and the
  -- INSERT fires `external_users_aliases_key` whenever the two rows share an
  -- alias set (common: one was produced from the same scraper source with a
  -- different hash, and the rows carry the same email/descriptor aliases).
  -- The losers are moved out of the `WHERE deleted_at IS NULL` partial index
  -- here; their alias union is still merged into the winners later in Step 5.
  UPDATE external_users SET deleted_at = NOW()
  FROM _eu_merges mp
  WHERE external_users.id = mp.loser_id
    AND external_users.deleted_at IS NULL;

  IF v_debug THEN
    GET DIAGNOSTICS v_row_count = ROW_COUNT;
    PERFORM _debug_log('step3a_presoft_delete_live_losers', jsonb_build_object('rows_affected', v_row_count));
  END IF;

  -- Step 3b: Pre-insert winners from temp table so FK remaps don't violate constraints
  EXECUTE format('
    INSERT INTO external_users (id, aliases, name, account_id, user_type, email, scraper_id, created_at, updated_at, created_by)
    SELECT id, aliases, name, account_id, user_type, email, scraper_id, created_at, updated_at, created_by
    FROM %I WHERE id IN (SELECT DISTINCT winner_id FROM _eu_merges)
    ON CONFLICT (id) DO NOTHING
  ', p_temp_table);

  IF v_debug THEN
    GET DIAGNOSTICS v_row_count = ROW_COUNT;
    PERFORM _debug_log('step3b_preinsert', jsonb_build_object('rows_affected', v_row_count));
  END IF;

  -- Step 4: Remap FKs in bulk without violating unique constraints
  CREATE TEMP TABLE _eu_ca_dups (id TEXT PRIMARY KEY) ON COMMIT DROP;
  INSERT INTO _eu_ca_dups (id)
  SELECT candidate.id
  FROM (
    SELECT ca.id,
           EXISTS (
             SELECT 1
             FROM config_access existing
             WHERE existing.deleted_at IS NULL
               AND existing.id <> ca.id
               AND existing.config_id = ca.config_id
               AND existing.external_user_id = mp.winner_id
               AND existing.external_group_id IS NOT DISTINCT FROM ca.external_group_id
               AND existing.external_role_id IS NOT DISTINCT FROM ca.external_role_id
           ) AS collides_with_live,
           ROW_NUMBER() OVER (
             PARTITION BY ca.config_id, mp.winner_id, ca.external_group_id, ca.external_role_id
             ORDER BY ca.created_at, ca.id
           ) AS target_rank
    FROM config_access ca
    JOIN _eu_merges mp ON ca.external_user_id = mp.loser_id
    WHERE ca.deleted_at IS NULL
  ) candidate
  WHERE candidate.collides_with_live OR candidate.target_rank > 1;

  UPDATE config_access
  SET deleted_at = NOW()
  WHERE deleted_at IS NULL
    AND id IN (SELECT id FROM _eu_ca_dups);

  UPDATE config_access
  SET external_user_id = mp.winner_id
  FROM _eu_merges mp
  WHERE config_access.external_user_id = mp.loser_id
    AND NOT EXISTS (SELECT 1 FROM _eu_ca_dups d WHERE d.id = config_access.id);

  UPDATE access_reviews SET external_user_id = mp.winner_id
  FROM _eu_merges mp WHERE access_reviews.external_user_id = mp.loser_id;

  CREATE TEMP TABLE _eu_log_agg ON COMMIT DROP AS
  SELECT cal.config_id,
         mp.winner_id AS external_user_id,
         cal.scraper_id,
         MAX(cal.created_at) AS created_at,
         COALESCE(SUM(COALESCE(cal.count, 1)), 0)::integer AS count,
         (ARRAY_AGG(cal.mfa ORDER BY cal.created_at DESC, cal.external_user_id::text))[1] AS mfa,
         (ARRAY_AGG(cal.properties ORDER BY cal.created_at DESC, cal.external_user_id::text))[1] AS properties
  FROM config_access_logs cal
  JOIN _eu_merges mp ON cal.external_user_id = mp.loser_id
  GROUP BY cal.config_id, mp.winner_id, cal.scraper_id;

  INSERT INTO config_access_logs (config_id, external_user_id, scraper_id, created_at, mfa, properties, count)
  SELECT config_id, external_user_id, scraper_id, created_at, mfa, properties, count
  FROM _eu_log_agg
  ON CONFLICT (config_id, external_user_id, scraper_id) DO UPDATE SET
    count = COALESCE(config_access_logs.count, 0) + COALESCE(EXCLUDED.count, 0),
    created_at = GREATEST(config_access_logs.created_at, EXCLUDED.created_at),
    mfa = CASE WHEN EXCLUDED.created_at >= config_access_logs.created_at THEN EXCLUDED.mfa ELSE config_access_logs.mfa END,
    properties = CASE WHEN EXCLUDED.created_at >= config_access_logs.created_at THEN EXCLUDED.properties ELSE config_access_logs.properties END;

  DELETE FROM config_access_logs USING _eu_merges mp
  WHERE config_access_logs.external_user_id = mp.loser_id;

  INSERT INTO external_user_groups (external_user_id, external_group_id, created_at)
  SELECT mp.winner_id, eug.external_group_id, eug.created_at
  FROM external_user_groups eug JOIN _eu_merges mp ON eug.external_user_id = mp.loser_id
  WHERE eug.deleted_at IS NULL
  ON CONFLICT (external_user_id, external_group_id) DO NOTHING;

  DELETE FROM external_user_groups USING _eu_merges mp
  WHERE external_user_groups.external_user_id = mp.loser_id;

  -- Step 5: Merge aliases from losers into winners in the live table.
  -- We also append `loser.id::text` to the winner's alias union so that
  -- future lookups by the loser's old id can recover the winner.
  -- entity.aliases never contains entity.id for live rows; loser ids only
  -- become aliases here, at the moment they stop being canonical.
  --
  -- A loser may live in the temp table (LEFT JOIN tmp), the live table
  -- (LEFT JOIN external_users) or both — we union all available alias
  -- sources together with the loser id itself.
  EXECUTE format('
    UPDATE external_users SET
      aliases = NULLIF(ARRAY(SELECT DISTINCT unnest FROM unnest(external_users.aliases || agg.all_aliases) ORDER BY 1), ''{}''::text[])
    FROM (
      SELECT mp.winner_id, array_agg(DISTINCT a) AS all_aliases
      FROM _eu_merges mp
      LEFT JOIN %1$I tmp_src ON tmp_src.id = mp.loser_id
      LEFT JOIN external_users live_src ON live_src.id = mp.loser_id
      CROSS JOIN LATERAL unnest(
        COALESCE(tmp_src.aliases, ''{}''::text[])
        || COALESCE(live_src.aliases, ''{}''::text[])
        || ARRAY[mp.loser_id::text]
      ) AS a
      GROUP BY mp.winner_id
    ) agg
    WHERE external_users.id = agg.winner_id
  ', p_temp_table);

  -- Step 6: Soft-delete losers
  UPDATE external_users SET deleted_at = NOW()
  FROM _eu_merges mp WHERE external_users.id = mp.loser_id;

  -- Step 7: Consolidate temp table (merge aliases from all losers - both
  -- temp and live - into temp winners). As in Step 5, also append the
  -- loser id itself so it remains lookupable via aliases after the merge.
  EXECUTE format('
    UPDATE %1$I t SET
      aliases = NULLIF(ARRAY(SELECT DISTINCT unnest FROM unnest(t.aliases || agg.all_aliases) ORDER BY 1), ''{}''::text[])
    FROM (
      SELECT mp.winner_id, array_agg(DISTINCT a) AS all_aliases
      FROM _eu_merges mp
      LEFT JOIN %1$I tmp_src ON tmp_src.id = mp.loser_id
      LEFT JOIN external_users live_src ON live_src.id = mp.loser_id
      CROSS JOIN LATERAL unnest(
        COALESCE(tmp_src.aliases, ''{}''::text[])
        || COALESCE(live_src.aliases, ''{}''::text[])
        || ARRAY[mp.loser_id::text]
      ) AS a
      GROUP BY mp.winner_id
    ) agg WHERE t.id = agg.winner_id
  ', p_temp_table);

  EXECUTE format('DELETE FROM %I USING _eu_merges mp WHERE id = mp.loser_id', p_temp_table);

  IF v_debug THEN
    EXECUTE format('SELECT count(*) FROM %I', p_temp_table) INTO v_row_count;
    PERFORM _debug_log('step7_post_delete_temp', jsonb_build_object('remaining_temp_rows', v_row_count));
  END IF;

  -- Step 8: Upsert survivors
  EXECUTE format('
    INSERT INTO external_users (id, aliases, name, account_id, user_type, email, scraper_id, created_at, updated_at, created_by)
    SELECT id, aliases, name, account_id, user_type, email, scraper_id, created_at, updated_at, created_by FROM %I
    ON CONFLICT (id) DO UPDATE SET
      aliases = NULLIF(ARRAY(SELECT DISTINCT unnest FROM unnest(external_users.aliases || EXCLUDED.aliases) ORDER BY 1), ''{}''::text[]),
      name = EXCLUDED.name, account_id = EXCLUDED.account_id,
      user_type = EXCLUDED.user_type, email = EXCLUDED.email,
      updated_at = EXCLUDED.updated_at, deleted_at = NULL
  ', p_temp_table);

  IF v_debug THEN
    GET DIAGNOSTICS v_row_count = ROW_COUNT;
    PERFORM _debug_log('step8_upsert', jsonb_build_object('rows_affected', v_row_count));
  END IF;

  RETURN QUERY SELECT mp.loser_id, mp.winner_id FROM _eu_merges mp;
END;
$$ LANGUAGE plpgsql;


CREATE OR REPLACE FUNCTION merge_and_upsert_external_groups(p_temp_table TEXT)
RETURNS TABLE(loser_id UUID, winner_id UUID) AS $$
BEGIN
  LOCK TABLE config_access, access_reviews, config_access_logs, external_user_groups,
    external_users, external_groups, external_roles IN SHARE ROW EXCLUSIVE MODE;

  -- Step 0: Normalize temp aliases to match the normalize_aliases() trigger
  -- on external_groups (lowercase + sorted + distinct). The edge-build below
  -- uses byte-exact && which would otherwise miss any temp row whose aliases
  -- differ from the normalized live row only by case. See the equivalent
  -- Step 0 in merge_and_upsert_external_users for details.
  EXECUTE format('
    UPDATE %1$I SET
      aliases = ARRAY(SELECT DISTINCT LOWER(elem) FROM unnest(aliases) AS elem ORDER BY LOWER(elem))
    WHERE aliases IS NOT NULL
  ', p_temp_table);

  EXECUTE format('
    CREATE TEMP TABLE _eg_edges ON COMMIT DROP AS
    SELECT DISTINCT a.id AS id1, b.id AS id2
    FROM %1$I a JOIN %1$I b ON a.id::text < b.id::text AND a.aliases && b.aliases
    UNION
    SELECT DISTINCT CASE WHEN tmp.id::text < live.id::text THEN tmp.id ELSE live.id END,
                    CASE WHEN tmp.id::text < live.id::text THEN live.id ELSE tmp.id END
    FROM %1$I tmp JOIN external_groups live
      ON tmp.aliases && live.aliases AND tmp.id != live.id AND live.deleted_at IS NULL
  ', p_temp_table);

  IF NOT EXISTS (SELECT 1 FROM _eg_edges) THEN
    EXECUTE format('
      INSERT INTO external_groups (id, aliases, name, account_id, scraper_id, group_type, created_at, updated_at)
      SELECT id, aliases, name, account_id, scraper_id, group_type, created_at, updated_at FROM %I
      ON CONFLICT (id) DO UPDATE SET
        aliases = NULLIF(ARRAY(SELECT DISTINCT unnest FROM unnest(external_groups.aliases || EXCLUDED.aliases) ORDER BY 1), ''{}''::text[]),
        name = EXCLUDED.name, account_id = EXCLUDED.account_id, group_type = EXCLUDED.group_type,
        updated_at = EXCLUDED.updated_at, deleted_at = NULL
    ', p_temp_table);
    RETURN;
  END IF;

  CREATE TEMP TABLE _eg_comp (node UUID PRIMARY KEY, leader UUID) ON COMMIT DROP;
  INSERT INTO _eg_comp (node, leader)
    SELECT DISTINCT id1, id1 FROM _eg_edges UNION SELECT DISTINCT id2, id2 FROM _eg_edges;

  LOOP
    UPDATE _eg_comp c SET leader = sub.new_leader::uuid
    FROM (
      SELECT c2.node, LEAST(c2.leader::text, MIN(e.min_neighbor::text)) AS new_leader
      FROM _eg_comp c2
      JOIN (
        SELECT id1, MIN(c3.leader::text)::uuid AS min_neighbor FROM _eg_edges JOIN _eg_comp c3 ON c3.node = id2 GROUP BY id1
        UNION ALL
        SELECT id2, MIN(c3.leader::text)::uuid AS min_neighbor FROM _eg_edges JOIN _eg_comp c3 ON c3.node = id1 GROUP BY id2
      ) e ON e.id1 = c2.node
      GROUP BY c2.node, c2.leader
      HAVING LEAST(c2.leader::text, MIN(e.min_neighbor::text)) < c2.leader::text
    ) sub WHERE c.node = sub.node;
    EXIT WHEN NOT FOUND;
  END LOOP;

  CREATE TEMP TABLE _eg_merges (loser_id UUID PRIMARY KEY, winner_id UUID) ON COMMIT DROP;
  INSERT INTO _eg_merges SELECT node, leader FROM _eg_comp WHERE node != leader;

  -- Step 3a: Pre-soft-delete live losers BEFORE pre-insert so the partial
  -- unique index doesn't fire on a winner with identical aliases to a
  -- still-active loser. See the equivalent Step 3a in
  -- merge_and_upsert_external_users for details.
  UPDATE external_groups SET deleted_at = NOW()
  FROM _eg_merges mp
  WHERE external_groups.id = mp.loser_id
    AND external_groups.deleted_at IS NULL;

  EXECUTE format('
    INSERT INTO external_groups (id, aliases, name, account_id, scraper_id, group_type, created_at, updated_at)
    SELECT id, aliases, name, account_id, scraper_id, group_type, created_at, updated_at
    FROM %I WHERE id IN (SELECT DISTINCT winner_id FROM _eg_merges)
    ON CONFLICT (id) DO NOTHING
  ', p_temp_table);

  CREATE TEMP TABLE _eg_ca_dups (id TEXT PRIMARY KEY) ON COMMIT DROP;
  INSERT INTO _eg_ca_dups (id)
  SELECT candidate.id
  FROM (
    SELECT ca.id,
           EXISTS (
             SELECT 1
             FROM config_access existing
             WHERE existing.deleted_at IS NULL
               AND existing.id <> ca.id
               AND existing.config_id = ca.config_id
               AND existing.external_user_id IS NOT DISTINCT FROM ca.external_user_id
               AND existing.external_group_id = mp.winner_id
               AND existing.external_role_id IS NOT DISTINCT FROM ca.external_role_id
           ) AS collides_with_live,
           ROW_NUMBER() OVER (
             PARTITION BY ca.config_id, ca.external_user_id, mp.winner_id, ca.external_role_id
             ORDER BY ca.created_at, ca.id
           ) AS target_rank
    FROM config_access ca
    JOIN _eg_merges mp ON ca.external_group_id = mp.loser_id
    WHERE ca.deleted_at IS NULL
  ) candidate
  WHERE candidate.collides_with_live OR candidate.target_rank > 1;

  UPDATE config_access
  SET deleted_at = NOW()
  WHERE deleted_at IS NULL
    AND id IN (SELECT id FROM _eg_ca_dups);

  UPDATE config_access
  SET external_group_id = mp.winner_id
  FROM _eg_merges mp
  WHERE config_access.external_group_id = mp.loser_id
    AND NOT EXISTS (SELECT 1 FROM _eg_ca_dups d WHERE d.id = config_access.id);

  INSERT INTO external_user_groups (external_user_id, external_group_id, created_at)
  SELECT eug.external_user_id, mp.winner_id, eug.created_at
  FROM external_user_groups eug JOIN _eg_merges mp ON eug.external_group_id = mp.loser_id
  WHERE eug.deleted_at IS NULL
  ON CONFLICT (external_user_id, external_group_id) DO NOTHING;

  DELETE FROM external_user_groups USING _eg_merges mp
  WHERE external_user_groups.external_group_id = mp.loser_id;

  -- Merge loser aliases (and the loser id itself) into the winner. See the
  -- equivalent step in merge_and_upsert_external_users for the reasoning,
  -- including why we LEFT JOIN both temp and live as alias sources.
  EXECUTE format('
    UPDATE external_groups SET
      aliases = NULLIF(ARRAY(SELECT DISTINCT unnest FROM unnest(external_groups.aliases || agg.all_aliases) ORDER BY 1), ''{}''::text[])
    FROM (
      SELECT mp.winner_id, array_agg(DISTINCT a) AS all_aliases
      FROM _eg_merges mp
      LEFT JOIN %1$I tmp_src ON tmp_src.id = mp.loser_id
      LEFT JOIN external_groups live_src ON live_src.id = mp.loser_id
      CROSS JOIN LATERAL unnest(
        COALESCE(tmp_src.aliases, ''{}''::text[])
        || COALESCE(live_src.aliases, ''{}''::text[])
        || ARRAY[mp.loser_id::text]
      ) AS a
      GROUP BY mp.winner_id
    ) agg WHERE external_groups.id = agg.winner_id
  ', p_temp_table);

  UPDATE external_groups SET deleted_at = NOW()
  FROM _eg_merges mp WHERE external_groups.id = mp.loser_id;

  EXECUTE format('
    UPDATE %1$I t SET
      aliases = NULLIF(ARRAY(SELECT DISTINCT unnest FROM unnest(t.aliases || agg.all_aliases) ORDER BY 1), ''{}''::text[])
    FROM (
      SELECT mp.winner_id, array_agg(DISTINCT a) AS all_aliases
      FROM _eg_merges mp
      LEFT JOIN %1$I tmp_src ON tmp_src.id = mp.loser_id
      LEFT JOIN external_groups live_src ON live_src.id = mp.loser_id
      CROSS JOIN LATERAL unnest(
        COALESCE(tmp_src.aliases, ''{}''::text[])
        || COALESCE(live_src.aliases, ''{}''::text[])
        || ARRAY[mp.loser_id::text]
      ) AS a
      GROUP BY mp.winner_id
    ) agg WHERE t.id = agg.winner_id
  ', p_temp_table);

  EXECUTE format('DELETE FROM %I USING _eg_merges mp WHERE id = mp.loser_id', p_temp_table);

  EXECUTE format('
    INSERT INTO external_groups (id, aliases, name, account_id, scraper_id, group_type, created_at, updated_at)
    SELECT id, aliases, name, account_id, scraper_id, group_type, created_at, updated_at FROM %I
    ON CONFLICT (id) DO UPDATE SET
      aliases = NULLIF(ARRAY(SELECT DISTINCT unnest FROM unnest(external_groups.aliases || EXCLUDED.aliases) ORDER BY 1), ''{}''::text[]),
      name = EXCLUDED.name, account_id = EXCLUDED.account_id, group_type = EXCLUDED.group_type,
      updated_at = EXCLUDED.updated_at, deleted_at = NULL
  ', p_temp_table);

  RETURN QUERY SELECT mp.loser_id, mp.winner_id FROM _eg_merges mp;
END;
$$ LANGUAGE plpgsql;


CREATE OR REPLACE FUNCTION merge_and_upsert_external_roles(p_temp_table TEXT)
RETURNS TABLE(loser_id UUID, winner_id UUID) AS $$
BEGIN
  LOCK TABLE config_access, access_reviews, config_access_logs, external_user_groups,
    external_users, external_groups, external_roles IN SHARE ROW EXCLUSIVE MODE;

  -- Step 0: Normalize temp aliases to match the normalize_aliases() trigger
  -- on external_roles. See Step 0 in merge_and_upsert_external_users.
  EXECUTE format('
    UPDATE %1$I SET
      aliases = ARRAY(SELECT DISTINCT LOWER(elem) FROM unnest(aliases) AS elem ORDER BY LOWER(elem))
    WHERE aliases IS NOT NULL
  ', p_temp_table);

  EXECUTE format('
    CREATE TEMP TABLE _er_edges ON COMMIT DROP AS
    SELECT DISTINCT a.id AS id1, b.id AS id2
    FROM %1$I a JOIN %1$I b ON a.id::text < b.id::text AND a.aliases && b.aliases
    UNION
    SELECT DISTINCT CASE WHEN tmp.id::text < live.id::text THEN tmp.id ELSE live.id END,
                    CASE WHEN tmp.id::text < live.id::text THEN live.id ELSE tmp.id END
    FROM %1$I tmp JOIN external_roles live
      ON tmp.aliases && live.aliases AND tmp.id != live.id AND live.deleted_at IS NULL
  ', p_temp_table);

  IF NOT EXISTS (SELECT 1 FROM _er_edges) THEN
    EXECUTE format('
      INSERT INTO external_roles (id, aliases, name, account_id, role_type, description, scraper_id, application_id, created_at, updated_at)
      SELECT id, aliases, name, account_id, role_type, description, scraper_id, application_id, created_at, updated_at FROM %I
      ON CONFLICT (id) DO UPDATE SET
        aliases = NULLIF(ARRAY(SELECT DISTINCT unnest FROM unnest(external_roles.aliases || EXCLUDED.aliases) ORDER BY 1), ''{}''::text[]),
        name = EXCLUDED.name, account_id = EXCLUDED.account_id,
        role_type = EXCLUDED.role_type, description = EXCLUDED.description,
        updated_at = EXCLUDED.updated_at, deleted_at = NULL
    ', p_temp_table);
    RETURN;
  END IF;

  CREATE TEMP TABLE _er_comp (node UUID PRIMARY KEY, leader UUID) ON COMMIT DROP;
  INSERT INTO _er_comp (node, leader)
    SELECT DISTINCT id1, id1 FROM _er_edges UNION SELECT DISTINCT id2, id2 FROM _er_edges;

  LOOP
    UPDATE _er_comp c SET leader = sub.new_leader::uuid
    FROM (
      SELECT c2.node, LEAST(c2.leader::text, MIN(e.min_neighbor::text)) AS new_leader
      FROM _er_comp c2
      JOIN (
        SELECT id1, MIN(c3.leader::text)::uuid AS min_neighbor FROM _er_edges JOIN _er_comp c3 ON c3.node = id2 GROUP BY id1
        UNION ALL
        SELECT id2, MIN(c3.leader::text)::uuid AS min_neighbor FROM _er_edges JOIN _er_comp c3 ON c3.node = id1 GROUP BY id2
      ) e ON e.id1 = c2.node
      GROUP BY c2.node, c2.leader
      HAVING LEAST(c2.leader::text, MIN(e.min_neighbor::text)) < c2.leader::text
    ) sub WHERE c.node = sub.node;
    EXIT WHEN NOT FOUND;
  END LOOP;

  CREATE TEMP TABLE _er_merges (loser_id UUID PRIMARY KEY, winner_id UUID) ON COMMIT DROP;
  INSERT INTO _er_merges SELECT node, leader FROM _er_comp WHERE node != leader;

  -- Step 3a: Pre-soft-delete live losers BEFORE pre-insert. See Step 3a in
  -- merge_and_upsert_external_users.
  UPDATE external_roles SET deleted_at = NOW()
  FROM _er_merges mp
  WHERE external_roles.id = mp.loser_id
    AND external_roles.deleted_at IS NULL;

  EXECUTE format('
    INSERT INTO external_roles (id, aliases, name, account_id, role_type, description, scraper_id, application_id, created_at, updated_at)
    SELECT id, aliases, name, account_id, role_type, description, scraper_id, application_id, created_at, updated_at
    FROM %I WHERE id IN (SELECT DISTINCT winner_id FROM _er_merges)
    ON CONFLICT (id) DO NOTHING
  ', p_temp_table);

  CREATE TEMP TABLE _er_ca_dups (id TEXT PRIMARY KEY) ON COMMIT DROP;
  INSERT INTO _er_ca_dups (id)
  SELECT candidate.id
  FROM (
    SELECT ca.id,
           EXISTS (
             SELECT 1
             FROM config_access existing
             WHERE existing.deleted_at IS NULL
               AND existing.id <> ca.id
               AND existing.config_id = ca.config_id
               AND existing.external_user_id IS NOT DISTINCT FROM ca.external_user_id
               AND existing.external_group_id IS NOT DISTINCT FROM ca.external_group_id
               AND existing.external_role_id = mp.winner_id
           ) AS collides_with_live,
           ROW_NUMBER() OVER (
             PARTITION BY ca.config_id, ca.external_user_id, ca.external_group_id, mp.winner_id
             ORDER BY ca.created_at, ca.id
           ) AS target_rank
    FROM config_access ca
    JOIN _er_merges mp ON ca.external_role_id = mp.loser_id
    WHERE ca.deleted_at IS NULL
  ) candidate
  WHERE candidate.collides_with_live OR candidate.target_rank > 1;

  UPDATE config_access
  SET deleted_at = NOW()
  WHERE deleted_at IS NULL
    AND id IN (SELECT id FROM _er_ca_dups);

  UPDATE config_access
  SET external_role_id = mp.winner_id
  FROM _er_merges mp
  WHERE config_access.external_role_id = mp.loser_id
    AND NOT EXISTS (SELECT 1 FROM _er_ca_dups d WHERE d.id = config_access.id);

  UPDATE access_reviews SET external_role_id = mp.winner_id
  FROM _er_merges mp WHERE access_reviews.external_role_id = mp.loser_id;

  -- Merge loser aliases (and the loser id itself) into the winner. See the
  -- equivalent step in merge_and_upsert_external_users for the reasoning,
  -- including why we LEFT JOIN both temp and live as alias sources.
  EXECUTE format('
    UPDATE external_roles SET
      aliases = NULLIF(ARRAY(SELECT DISTINCT unnest FROM unnest(external_roles.aliases || agg.all_aliases) ORDER BY 1), ''{}''::text[])
    FROM (
      SELECT mp.winner_id, array_agg(DISTINCT a) AS all_aliases
      FROM _er_merges mp
      LEFT JOIN %1$I tmp_src ON tmp_src.id = mp.loser_id
      LEFT JOIN external_roles live_src ON live_src.id = mp.loser_id
      CROSS JOIN LATERAL unnest(
        COALESCE(tmp_src.aliases, ''{}''::text[])
        || COALESCE(live_src.aliases, ''{}''::text[])
        || ARRAY[mp.loser_id::text]
      ) AS a
      GROUP BY mp.winner_id
    ) agg WHERE external_roles.id = agg.winner_id
  ', p_temp_table);

  UPDATE external_roles SET deleted_at = NOW()
  FROM _er_merges mp WHERE external_roles.id = mp.loser_id;

  EXECUTE format('
    UPDATE %1$I t SET
      aliases = NULLIF(ARRAY(SELECT DISTINCT unnest FROM unnest(t.aliases || agg.all_aliases) ORDER BY 1), ''{}''::text[])
    FROM (
      SELECT mp.winner_id, array_agg(DISTINCT a) AS all_aliases
      FROM _er_merges mp
      LEFT JOIN %1$I tmp_src ON tmp_src.id = mp.loser_id
      LEFT JOIN external_roles live_src ON live_src.id = mp.loser_id
      CROSS JOIN LATERAL unnest(
        COALESCE(tmp_src.aliases, ''{}''::text[])
        || COALESCE(live_src.aliases, ''{}''::text[])
        || ARRAY[mp.loser_id::text]
      ) AS a
      GROUP BY mp.winner_id
    ) agg WHERE t.id = agg.winner_id
  ', p_temp_table);

  EXECUTE format('DELETE FROM %I USING _er_merges mp WHERE id = mp.loser_id', p_temp_table);

  EXECUTE format('
    INSERT INTO external_roles (id, aliases, name, account_id, role_type, description, scraper_id, application_id, created_at, updated_at)
    SELECT id, aliases, name, account_id, role_type, description, scraper_id, application_id, created_at, updated_at FROM %I
    ON CONFLICT (id) DO UPDATE SET
      aliases = NULLIF(ARRAY(SELECT DISTINCT unnest FROM unnest(external_roles.aliases || EXCLUDED.aliases) ORDER BY 1), ''{}''::text[]),
      name = EXCLUDED.name, account_id = EXCLUDED.account_id,
      role_type = EXCLUDED.role_type, description = EXCLUDED.description,
      updated_at = EXCLUDED.updated_at, deleted_at = NULL
  ', p_temp_table);

  RETURN QUERY SELECT mp.loser_id, mp.winner_id FROM _er_merges mp;
END;
$$ LANGUAGE plpgsql;

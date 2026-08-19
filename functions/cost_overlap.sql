-- Bucket and window helpers for config cost.

-- Fraction of a cost row's charge period that falls inside a reporting window.
-- Rows coarser than the window straddle it, so rollups prorate with this rather than
-- including or excluding the whole row.
--
-- Both ranges are half-open [start, end). A row fully inside the window returns 1 and one
-- fully outside returns 0, so the common case stays exact.
CREATE OR REPLACE FUNCTION cost_window_overlap(
  p_start timestamptz, p_end timestamptz, w_start timestamptz, w_end timestamptz
) RETURNS numeric LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
  SELECT GREATEST(0, EXTRACT(epoch FROM (LEAST(p_end, w_end) - GREATEST(p_start, w_start))))
       / NULLIF(EXTRACT(epoch FROM (p_end - p_start)), 0);
$$;

-- Start of the bucket of the given width containing ts, anchored on the Unix epoch.
--
-- The width is a parameter rather than a property lookup so this stays IMMUTABLE, which is
-- what lets it be used from a materialized view and an index. The compaction job reads the
-- configured level widths and passes them in.
--
-- Anchoring on the epoch makes the common widths fall on natural boundaries — 3600 gives
-- clock hours, 86400 gives UTC midnights — while any width still tiles the timeline
-- without gaps, so a finer bucket always nests inside exactly one coarser bucket provided
-- the widths divide each other.
CREATE OR REPLACE FUNCTION cost_bucket(ts timestamptz, width_seconds bigint)
RETURNS timestamptz LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
  SELECT to_timestamp(floor(EXTRACT(epoch FROM ts) / width_seconds) * width_seconds);
$$;

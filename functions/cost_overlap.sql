-- Fraction of a cost row's charge period that falls inside a reporting window.
-- Week- and month-grain rows straddle the 1d/7d/30d windows, so rollups prorate
-- with this rather than including or excluding the whole row.
--
-- Both ranges are half-open [start, end). A day-grain row fully inside the window
-- returns 1 and one fully outside returns 0, so the common case stays exact.
CREATE OR REPLACE FUNCTION cost_window_overlap(
  p_start timestamptz, p_end timestamptz, w_start timestamptz, w_end timestamptz
) RETURNS numeric LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
  SELECT GREATEST(0, EXTRACT(epoch FROM (LEAST(p_end, w_end) - GREATEST(p_start, w_start))))
       / NULLIF(EXTRACT(epoch FROM (p_end - p_start)), 0);
$$;

CREATE OR REPLACE FUNCTION permissions_for_obj_selector(
  p_field TEXT,
  p_name TEXT,
  p_namespace TEXT DEFAULT NULL
)
RETURNS SETOF permissions_summary
LANGUAGE plpgsql
STABLE
AS $$
BEGIN
  IF p_field IS NULL OR trim(p_field) = '' THEN
    RAISE EXCEPTION 'p_field is required';
  END IF;

  IF p_name IS NULL OR trim(p_name) = '' THEN
    RAISE EXCEPTION 'p_name is required';
  END IF;

  RETURN QUERY
  SELECT ps.*
  FROM permissions_summary ps
  WHERE COALESCE(ps.object_selector, '{}'::jsonb) ? p_field
    AND EXISTS (
      SELECT 1
      FROM jsonb_array_elements(
        COALESCE(ps.object_selector -> p_field, '[]'::jsonb)
      ) AS selector
      WHERE selector ->> 'name' = p_name
        AND (
          p_namespace IS NULL
          OR trim(p_namespace) = ''
          OR COALESCE(selector ->> 'namespace', '') = ''
          OR selector ->> 'namespace' = p_namespace
        )
    );
END;
$$;


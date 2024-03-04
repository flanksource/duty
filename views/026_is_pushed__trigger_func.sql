CREATE
OR REPLACE FUNCTION reset_is_pushed_before_update() RETURNS TRIGGER AS $$
BEGIN
  -- If any column other than is_pushed is changed, reset is_pushed to false.
  IF NEW IS DISTINCT FROM OLD AND NEW.is_pushed IS NOT DISTINCT FROM OLD.is_pushed THEN
    NEW.is_pushed = false;
  END IF;

  RETURN NEW;
END
$$ LANGUAGE plpgsql;
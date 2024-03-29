CREATE SEQUENCE IF NOT EXISTS incident_id_sequence INCREMENT 1 START 1;

CREATE OR REPLACE FUNCTION format_incident_id(seq_value BIGINT) 
RETURNS VARCHAR
AS $$
  DECLARE
    result VARCHAR;
  BEGIN
    RETURN 'INC-' || seq_value::TEXT;
  END;
$$ 
LANGUAGE plpgsql;

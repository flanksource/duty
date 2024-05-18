CREATE OR REPLACE FUNCTION populate_config_item_name() 
RETURNS TRIGGER AS $$
BEGIN
  IF NEW.name IS NULL OR new.NAME = '' THEN
    NEW.name = RIGHT(NEW.id::TEXT, 12);
  END IF;
    
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER check_config_item_name
BEFORE INSERT ON config_items
FOR EACH ROW
EXECUTE PROCEDURE populate_config_item_name();

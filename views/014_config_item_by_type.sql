CREATE OR REPLACE VIEW config_items_aws AS
    SELECT id, external_id, config_type, external_type, name, account, tags, tags->>'zone' as zone, tags->>'region' as region
    FROM config_items
    WHERE external_type LIKE 'AWS::%';

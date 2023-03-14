CREATE OR REPLACE VIEW config_items_aws AS
    SELECT *, tags->>'zone' as zone, tags->>'region' as region
    FROM config_items
    WHERE external_type LIKE 'AWS::%';

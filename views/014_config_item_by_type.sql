CREATE OR REPLACE VIEW config_items_aws AS
    SELECT *, tags->>'zone' AS zone, tags->>'region' AS region, tags->>'account' AS account
    FROM configs
    WHERE external_type LIKE 'AWS::%';

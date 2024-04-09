CREATE OR REPLACE VIEW
  config_items_aws AS
SELECT
  *,
  labels ->> 'zone' AS zone,
  labels ->> 'region' AS region,
  labels ->> 'account' AS account
FROM
  configs
WHERE
  type LIKE 'AWS::%';

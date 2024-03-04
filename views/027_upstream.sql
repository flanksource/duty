CREATE OR REPLACE VIEW unpushed_tables 
AS
SELECT
  c.relname
FROM
  pg_index i
  JOIN pg_class c ON c.oid = i.indrelid
  JOIN pg_class ic ON i.indexrelid = ic.oid
WHERE
  i.indexrelid :: regclass :: TEXT LIKE '%_is_pushed_idx'
  AND ic.reltuples > 0
-- Spend the scraper could not attach to a config item. Retained rather than folded into
-- the account total so the gap is visible and the compaction job can re-attach it once
-- the config item appears.
DROP VIEW IF EXISTS config_costs_unmatched;
CREATE OR REPLACE VIEW config_costs_unmatched AS
  SELECT
    source_key,
    source_record_id,
    scraper_id,
    external_id,
    external_config_type,
    external_config_scraper_id,
    external_config_labels,
    billing_currency,
    SUM(effective_cost) AS effective_cost,
    SUM(billed_cost) AS billed_cost,
    MIN(period_start) AS since,
    MAX(period_end) AS until
  FROM config_costs
  WHERE config_id IS NULL
  GROUP BY source_key, source_record_id, scraper_id, external_id, external_config_type,
           external_config_scraper_id, external_config_labels, billing_currency;

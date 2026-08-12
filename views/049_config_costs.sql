-- dependsOn: functions/cost_overlap.sql, views/006_config_views.sql

-- Spend the scraper could not attach to a config item. Retained rather than folded into
-- the account total so the gap is visible and the compaction job can re-attach it once
-- the config item appears.
DROP VIEW IF EXISTS config_costs_unmatched;
CREATE OR REPLACE VIEW config_costs_unmatched AS
  SELECT
    external_id,
    scraper_id,
    billing_currency,
    SUM(effective_cost) AS effective_cost,
    SUM(billed_cost) AS billed_cost,
    MIN(period_start) AS since,
    MAX(period_end) AS until
  FROM config_costs
  WHERE config_id IS NULL
  GROUP BY external_id, scraper_id, billing_currency;

-- Populate missing config_items into config_items_last_scraped_time table
INSERT INTO config_items_last_scraped_time (config_id, last_scraped_time)
SELECT ci.id, NOW()
FROM config_items ci
LEFT JOIN config_items_last_scraped_time cilst ON cilst.config_id = ci.id
WHERE cilst.config_id IS NULL;

--
UPDATE checks_unlogged
SET last_runtime = NOW()
WHERE EXTRACT(YEAR FROM last_runtime) = 1;

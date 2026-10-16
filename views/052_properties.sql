-- dependsOn: functions/property_type.sql

-- Recompute stored types when the public key list changes.
UPDATE properties SET name = name WHERE type IS DISTINCT FROM property_type(name);

CREATE OR REPLACE VIEW feature_flags AS
SELECT name, value, type, created_by, created_at, updated_at, deleted_at
FROM properties
WHERE type = 'feature_flag' AND deleted_at IS NULL;

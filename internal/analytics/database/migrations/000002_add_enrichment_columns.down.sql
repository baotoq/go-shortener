DROP INDEX IF EXISTS idx_clicks_summary;
DROP INDEX IF EXISTS idx_clicks_source;
DROP INDEX IF EXISTS idx_clicks_device;
DROP INDEX IF EXISTS idx_clicks_country;
DROP INDEX IF EXISTS idx_clicks_time_range;

ALTER TABLE clicks DROP COLUMN traffic_source;
ALTER TABLE clicks DROP COLUMN device_type;
ALTER TABLE clicks DROP COLUMN country_code;

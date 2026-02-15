ALTER TABLE clicks
  DROP COLUMN IF EXISTS country_code,
  DROP COLUMN IF EXISTS device_type,
  DROP COLUMN IF EXISTS traffic_source;

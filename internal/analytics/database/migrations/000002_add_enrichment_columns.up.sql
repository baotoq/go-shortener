ALTER TABLE clicks ADD COLUMN country_code TEXT NOT NULL DEFAULT 'Unknown';
ALTER TABLE clicks ADD COLUMN device_type TEXT NOT NULL DEFAULT 'Unknown';
ALTER TABLE clicks ADD COLUMN traffic_source TEXT NOT NULL DEFAULT 'Direct';

CREATE INDEX idx_clicks_time_range ON clicks(short_code, clicked_at);
CREATE INDEX idx_clicks_country ON clicks(short_code, country_code);
CREATE INDEX idx_clicks_device ON clicks(short_code, device_type);
CREATE INDEX idx_clicks_source ON clicks(short_code, traffic_source);
CREATE INDEX idx_clicks_summary ON clicks(short_code, clicked_at, country_code, device_type, traffic_source);

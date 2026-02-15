CREATE TABLE clicks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    short_code TEXT NOT NULL,
    clicked_at INTEGER NOT NULL,
    country_code TEXT NOT NULL,
    device_type TEXT NOT NULL,
    traffic_source TEXT NOT NULL
);

CREATE TABLE clicks (
  id          UUID PRIMARY KEY,
  short_code  VARCHAR(8) NOT NULL,
  clicked_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_clicks_short_code ON clicks (short_code);

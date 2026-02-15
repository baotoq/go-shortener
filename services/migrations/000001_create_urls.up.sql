CREATE TABLE urls (
  id           UUID PRIMARY KEY,
  short_code   VARCHAR(8) NOT NULL UNIQUE,
  original_url TEXT NOT NULL,
  click_count  BIGINT NOT NULL DEFAULT 0,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
  id TEXT NOT NULL,
  email TEXT NOT NULL,
  display_name TEXT NOT NULL,
  access_token TEXT NOT NULL,
  refresh_token TEXT NOT NULL,
  token_type TEXT,
  expiry TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id)
)

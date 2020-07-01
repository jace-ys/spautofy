CREATE TABLE IF NOT EXISTS accounts (
  user_id TEXT NOT NULL,
  schedule TEXT NOT NULL,
  track_limit INTEGER NOT NULL,
  with_confirm BOOLEAN,
  created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id),
  FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
)

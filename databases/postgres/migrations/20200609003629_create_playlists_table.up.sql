CREATE TABLE IF NOT EXISTS playlists (
  id TEXT UNIQUE,
  name TEXT UNIQUE NOT NULL,
  user_id TEXT NOT NULL,
  description TEXT NOT NULL,
  tracks TEXT[] NOT NULL,
  spotify_url TEXT,
  snapshot_id TEXT,
  created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (name),
  FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
)

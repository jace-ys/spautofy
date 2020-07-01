ALTER TABLE playlists DROP CONSTRAINT playlists_pkey;
ALTER TABLE playlists ADD PRIMARY KEY (name);

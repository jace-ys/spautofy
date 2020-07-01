ALTER TABLE schedules DROP CONSTRAINT schedules_pkey;
ALTER TABLE schedules ADD PRIMARY KEY (user_id);

CREATE TABLE IF NOT EXISTS reminder_schedule(item_id TEXT PRIMARY KEY, generation TEXT NOT NULL, next_at INTEGER NOT NULL, nag_minutes INTEGER NOT NULL, state TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS reminder_occurrence(id TEXT PRIMARY KEY, item_id TEXT NOT NULL, generation TEXT NOT NULL, state TEXT NOT NULL, entry_id TEXT NOT NULL, result_id TEXT NOT NULL DEFAULT '');
CREATE TABLE IF NOT EXISTS push_device(id TEXT PRIMARY KEY, token TEXT NOT NULL UNIQUE);
CREATE TABLE IF NOT EXISTS push_delivery(occurrence_id TEXT NOT NULL, device_id TEXT NOT NULL, next_at INTEGER NOT NULL, sent INTEGER NOT NULL DEFAULT 0, PRIMARY KEY(occurrence_id,device_id));

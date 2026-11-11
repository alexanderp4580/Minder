CREATE TABLE IF NOT EXISTS item(id TEXT PRIMARY KEY, body TEXT NOT NULL CHECK(json_valid(body))
 CHECK(json_extract(body,'$.status') IN ('open','in_progress','done','cancelled'))
 CHECK(json_extract(body,'$.version') >= 1)
 CHECK(length(trim(json_extract(body,'$.title'))) BETWEEN 1 AND 200));
 CREATE TABLE IF NOT EXISTS pending_change(id TEXT PRIMARY KEY, body TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS audit(id TEXT PRIMARY KEY, actor TEXT NOT NULL CHECK(actor IN ('owner','agent')), body TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS chat_entry(seq INTEGER PRIMARY KEY AUTOINCREMENT,id TEXT UNIQUE NOT NULL,body TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS event(cursor INTEGER PRIMARY KEY AUTOINCREMENT,entry_id TEXT NOT NULL,body TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS usage(id TEXT PRIMARY KEY,month TEXT NOT NULL,cost_eur REAL NOT NULL,body TEXT NOT NULL);
 PRAGMA user_version=1;
CREATE TABLE IF NOT EXISTS reminder_schedule(item_id TEXT PRIMARY KEY, generation TEXT NOT NULL, next_at INTEGER NOT NULL, nag_minutes INTEGER NOT NULL, state TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS reminder_occurrence(id TEXT PRIMARY KEY, item_id TEXT NOT NULL, generation TEXT NOT NULL, state TEXT NOT NULL, entry_id TEXT NOT NULL, result_id TEXT NOT NULL DEFAULT '');
CREATE TABLE IF NOT EXISTS push_device(id TEXT PRIMARY KEY, token TEXT NOT NULL UNIQUE);
CREATE TABLE IF NOT EXISTS push_delivery(occurrence_id TEXT NOT NULL, device_id TEXT NOT NULL, next_at INTEGER NOT NULL, sent INTEGER NOT NULL DEFAULT 0, PRIMARY KEY(occurrence_id,device_id));
PRAGMA user_version=2;

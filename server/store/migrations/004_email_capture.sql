CREATE TABLE capture(id TEXT PRIMARY KEY, body TEXT NOT NULL CHECK(json_valid(body)));
CREATE TABLE capture_receipt(id TEXT PRIMARY KEY, body TEXT NOT NULL CHECK(json_valid(body)));
CREATE TABLE item_source(item_id TEXT NOT NULL, source_id TEXT NOT NULL REFERENCES capture(id), PRIMARY KEY(item_id,source_id));
CREATE INDEX capture_jobs ON capture(json_extract(body,'$.status'),json_extract(body,'$.next_at'));

CREATE TABLE agent_turn(id TEXT PRIMARY KEY, body TEXT NOT NULL);
CREATE TABLE source(id TEXT PRIMARY KEY, body TEXT NOT NULL);
INSERT INTO source(id,body) SELECT id, json_object('id',id,'kind','email','entry_id',coalesce(json_extract(body,'$.entry_id'),''),'capture_id',id,'label',coalesce(json_extract(body,'$.subject'),''),'sender',coalesce(json_extract(body,'$.sender'),''),'created_at',coalesce(json_extract(body,'$.received_at'),''),'deleted',json(CASE WHEN coalesce(json_extract(body,'$.deleted'),0)=1 THEN 'true' ELSE 'false' END)) FROM capture;

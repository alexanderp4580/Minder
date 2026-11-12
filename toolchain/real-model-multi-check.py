#!/usr/bin/env python3
"""Check that the real model proposes two changes for one message, several times.

Needs .local/minder (built server) and a key in OPENROUTER_MINDER_TOKEN, for example
`tokencrypt run --with openrouter_minder_token -- python3 toolchain/real-model-multi-check.py`.
Uses a temporary database with two made-up items. Nothing is applied. Costs about 0.001 USD a run.
"""
import json, os, socket, sqlite3, subprocess, sys, tempfile, time, urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
RUNS = int(sys.argv[1]) if len(sys.argv) > 1 else 3
MESSAGE = "Move the dentist to Monday at 10 and lunch with Sam to Sunday at 13."
ITEMS = [("i1", "Dentist", "2026-11-13T08:00:00Z"), ("i2", "Lunch with Sam", "2026-11-14T11:00:00Z")]
EXPECTED = {"i1": "2026-11-16T09:00:00Z", "i2": "2026-11-15T12:00:00Z"}  # Monday 10:00 and Sunday 13:00, Amsterdam


def once() -> None:
    with socket.socket() as s:
        s.bind(("127.0.0.1", 0))
        port = s.getsockname()[1]
    fd, db = tempfile.mkstemp(prefix="multi-", suffix=".db", dir=ROOT / ".local")
    os.close(fd)
    env = dict(os.environ, OPENROUTER_API_KEY=os.environ["OPENROUTER_MINDER_TOKEN"], ENV="development",
               AUTH_MODE="dev", DATABASE_PATH=db, LISTEN_ADDR=f"127.0.0.1:{port}")
    env.pop("OPENROUTER_MINDER_TOKEN")
    env.pop("OPENROUTER_URL", None)
    server = subprocess.Popen([str(ROOT / ".local/minder")], env=env, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    base = f"http://127.0.0.1:{port}"
    get = lambda path: json.load(urllib.request.urlopen(base + path, timeout=30))
    try:
        for _ in range(50):
            try:
                get("/v1/health")
                break
            except Exception:
                time.sleep(0.1)
        conn = sqlite3.connect(db)
        for item_id, title, due in ITEMS:
            body = {"id": item_id, "title": title, "status": "open", "due_at": due, "timezone": "Europe/Amsterdam",
                    "version": 1, "created_at": "2026-11-10T00:00:00Z", "updated_at": "2026-11-10T00:00:00Z"}
            conn.execute("INSERT INTO item(id,body) VALUES(?,?)", (item_id, json.dumps(body)))
        conn.commit()
        request = urllib.request.Request(base + "/v1/chat/messages", method="POST", data=json.dumps({"text": MESSAGE}).encode(),
                                         headers={"Content-Type": "application/json"})
        urllib.request.urlopen(request, timeout=30)
        for _ in range(90):
            time.sleep(1)
            replies = [e for e in get("/v1/chat?after=0&limit=50") if e["kind"] != "user"]
            if replies:
                time.sleep(2)
                replies = [e for e in get("/v1/chat?after=0&limit=50") if e["kind"] != "user"]
                break
        kinds = [e["kind"] for e in replies]
        assert kinds == ["preview", "preview"], f"expected two previews, got {kinds}: {[e['text'] for e in replies]}"
        due = sorted(f["new"] for e in replies for f in e["change"]["fields"] if f["name"] == "due_at")
        assert due == sorted(EXPECTED.values()), f"wrong times: {due}"
    finally:
        server.terminate()
        os.remove(db)


for n in range(1, RUNS + 1):
    once()
    print(f"run {n}: two previews with the right times")
print(f"PASS: {RUNS} of {RUNS} runs")

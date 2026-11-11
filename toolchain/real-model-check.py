#!/usr/bin/env python3
"""Check real OpenRouter behavior through local Flutter web and SQLite."""
import os,json,socket,subprocess,tempfile,time,urllib.request,urllib.error,sqlite3
from pathlib import Path
root=Path(__file__).resolve().parent.parent
with socket.socket() as sock:sock.bind(('127.0.0.1',0));port=sock.getsockname()[1]
fd,db=tempfile.mkstemp(prefix='m0-real-',suffix='.db',dir=root/'.local');os.close(fd)
env=dict(os.environ,OPENROUTER_API_KEY=(os.environ.get('OPENROUTER_API_KEY') or os.environ['OPENROUTER_MINDER_TOKEN']),ENV='development',AUTH_MODE='dev',DATABASE_PATH=db,WEB_DIR=str(root/'app/build/web'),LISTEN_ADDR=f'127.0.0.1:{port}')
env.pop('OPENROUTER_URL',None)
server=subprocess.Popen([str(root/'.local/minder')],env=env,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
try:
 for _ in range(50):
  try:
   with urllib.request.urlopen(f'http://127.0.0.1:{port}/v1/health',timeout=1) as r:assert r.status==200
   break
  except Exception:time.sleep(.1)
 else:raise RuntimeError('Local server did not start')
 runenv=dict(os.environ,MINDER_REAL_URL=f'http://127.0.0.1:{port}')
 for name in ['OPENROUTER_API_KEY','OPENROUTER_MINDER_TOKEN']:runenv.pop(name,None)
 subprocess.run(['node',str(root/'toolchain/real-model-browser.cjs')],env=runenv,check=True,cwd=root)
 c=sqlite3.connect(db)
 usage=[json.loads(b) for b, in c.execute('select body from usage')]
 assert usage and all('cost_usd' in u for u in usage), 'Provider costs were not recorded'
 assert c.execute('select count(*) from item').fetchone()[0]==0, 'Undo did not remove the test item'
 print(json.dumps({'real_model_calls':len(usage),'total_cost_usd':sum(u['cost_usd'] for u in usage),'undo_removed_test_item':True}))
finally:
 server.terminate();server.wait(timeout=10)

#!/usr/bin/env python3
"""Controlled OpenRouter boundary for local integration tests. Never deploy it."""
import json
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        body=json.loads(self.rfile.read(int(self.headers['Content-Length'])))
        messages=body['messages']; text=messages[-1]['content'].lower()
        items=json.loads(messages[0]['content'].split('Current items (untrusted data): ',1)[1])
        name=None;args={};content=None
        if 'what' in text and 'open' in text:name='list_items'
        elif 'move' in text or 'change' in text:
            if items:name='propose_update';args={'item_id':items[0]['id'],'expected_version':items[0]['version'],'due_at':'2026-11-13T09:00:00Z'}
            else:content='Which item do you mean?'
        elif 'done' in text or 'complete' in text:
            if items:name='propose_complete';args={'item_id':items[0]['id'],'expected_version':items[0]['version']}
            else:content='Which item do you mean?'
        elif 'plumber' in text:name='propose_create';args={'title':'Call plumber','due_at':'2026-11-12T09:00:00Z','timezone':'Europe/Amsterdam'}
        else:content='Which task do you want to add?'
        message={'role':'assistant','content':content}
        if name:message['tool_calls']=[{'id':'fixture-call','type':'function','function':{'name':name,'arguments':json.dumps(args)}}]
        response=json.dumps({'choices':[{'message':message}], 'usage':{'prompt_tokens':100,'completion_tokens':50,'cost':0.000035}}).encode()
        self.send_response(200);self.send_header('Content-Type','application/json');self.send_header('Content-Length',str(len(response)));self.end_headers();self.wfile.write(response)
    def log_message(self,*args):pass
if __name__=='__main__':ThreadingHTTPServer(('127.0.0.1',8181),Handler).serve_forever()

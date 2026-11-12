#!/usr/bin/env python3
"""Controlled OpenRouter boundary for local integration tests. Never deploy it."""
import json
import os
from datetime import datetime, timedelta, timezone
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        body=json.loads(self.rfile.read(int(self.headers['Content-Length'])))
        messages=body['messages']
        if body.get('tools',[{}])[0].get('function',{}).get('name')=='extract_capture':
            return self.capture_response(messages)
        text=messages[-1]['content'].lower()
        items=json.loads(messages[0]['content'].split('Current items (untrusted data): ',1)[1])
        name=None;args={};content=None
        if 'what' in text and 'open' in text:name='list_items'
        elif 'move' in text or 'change' in text:
            if items:name='propose_update';args={'item_id':items[0]['id'],'expected_version':items[0]['version'],'due_at':'2026-11-13T09:00:00Z'}
            else:content='Which item do you mean?'
        elif 'done' in text or 'complete' in text:
            if items:name='propose_complete';args={'item_id':items[0]['id'],'expected_version':items[0]['version']}
            else:content='Which item do you mean?'
        elif 'remind' in text:name='propose_create';args={'title':'Water plants','remind_at':(datetime.now(timezone.utc)+timedelta(seconds=3)).strftime('%Y-%m-%dT%H:%M:%SZ'),'nag_minutes':15,'timezone':'Europe/Amsterdam'}
        elif 'plumber' in text:name='propose_create';args={'title':'Call plumber','due_at':'2026-11-12T09:00:00Z','timezone':'Europe/Amsterdam'}
        else:content='Which task do you want to add?'
        message={'role':'assistant','content':content}
        if name:message['tool_calls']=[{'id':'fixture-call','type':'function','function':{'name':name,'arguments':json.dumps(args)}}]
        response=json.dumps({'choices':[{'message':message}], 'usage':{'prompt_tokens':100,'completion_tokens':50,'cost':0.000035}}).encode()
        self.send_response(200);self.send_header('Content-Type','application/json');self.send_header('Content-Length',str(len(response)));self.end_headers();self.wfile.write(response)
    def capture_response(self,messages):
        source=json.loads(messages[-1]['content'][0]['text'].split('\nPreviously extracted candidates:',1)[0])
        document=source['source']; parts=document['parts']
        part=next((p for p in parts if p.get('text','').strip()),{'id':'1','text':'Booking'})
        text=part['text']; candidates=[]; answers=source.get('answers') or []
        evidence=[{'part':part['id'],'excerpt':text[:120]}]
        if 'cancel' in text.lower():
            items=source.get('existing_items') or []
            if items:
                item=next((item for item in items if item['title']=='Hotel booking'),items[0])
                candidates=[{'id':'cancel','proposal':{'action':'cancel_item','item_id':item['id'],'expected_version':item['version'],'title':item['title']},'evidence':evidence}]
        else:
            candidates=[{'id':'hotel','proposal':{'action':'create_item','kind':'event','title':'Hotel booking','start_date':'2026-11-12','end_date':'2026-11-15','notes':'Booking ABC123'},'evidence':evidence}]
            flight={'id':'return-flight','proposal':{'action':'create_item','kind':'event','title':'Return flight'},'question':'What date is your return flight?','evidence':evidence}
            if answers:
                flight['question']='';flight['proposal'].update({'start_at':'2026-11-21T11:00:00Z','end_at':'2026-11-21T19:00:00Z','start_timezone':'Europe/Amsterdam','end_timezone':'America/New_York'})
            candidates.append(flight)
        response=json.dumps({'choices':[{'finish_reason':'stop','message':{'content':None,'tool_calls':[{'id':'capture-fixture','type':'function','function':{'name':'extract_capture','arguments':json.dumps({'candidates':candidates,'questions':[]})}}]}}],'usage':{'cost':0.000035}}).encode()
        self.send_response(200);self.send_header('Content-Type','application/json');self.send_header('Content-Length',str(len(response)));self.end_headers();self.wfile.write(response)
    def log_message(self,*args):pass
if __name__=='__main__':ThreadingHTTPServer(('127.0.0.1',int(os.environ.get('MODEL_FIXTURE_PORT','8181'))),Handler).serve_forever()

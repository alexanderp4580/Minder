#!/usr/bin/env python3
"""Controlled OpenRouter boundary for local integration tests. Never deploy it."""
import json
import os
import threading
from datetime import datetime, timedelta, timezone
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

def text_of(message):
    content=message['content']
    return (content if isinstance(content,str) else json.dumps(content)).lower()

answer_gate=threading.Event()
answer_gate.set()

class Handler(BaseHTTPRequestHandler):
    def reply(self,content,name,args):
        message={'role':'assistant','content':content,'tool_calls':[{'id':'fixture-call','type':'function','function':{'name':name,'arguments':json.dumps(args)}}]}
        response=json.dumps({'choices':[{'finish_reason':'stop','message':message}],'usage':{'prompt_tokens':100,'completion_tokens':50,'cost':0.000035}}).encode()
        self.send_response(200);self.send_header('Content-Type','application/json');self.send_header('Content-Length',str(len(response)));self.end_headers();self.wfile.write(response)
    def do_POST(self):
        if self.path in ('/pause-capture-answers','/resume-capture-answers'):
            answer_gate.clear() if self.path=='/pause-capture-answers' else answer_gate.set()
            self.send_response(200);self.end_headers();return
        body=json.loads(self.rfile.read(int(self.headers['Content-Length'])))
        messages=body['messages']
        if body.get('tools',[{}])[0].get('function',{}).get('name')=='extract_capture':
            return self.capture_response(messages)
        text=messages[-1]['content'].lower()
        if 'lookup results' in text_of(messages[-1]):
            return self.reply(None,'propose_create',{'title':'Aviad at SnowWorld','location':'Witte Wereld 1, 6372 VG Landgraaf, Netherlands','website':'https://www.snowworld.com/landgraaf','start_date':'2026-11-17','end_date':'2026-11-24'})
        if 'snowworld' in text and any(t.get('function',{}).get('name')=='offer_lookup' for t in body.get('tools',[])):
            return self.reply(None,'offer_lookup',{'kind':'maps','query':'SnowWorld Landgraaf'})
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
        if answers: answer_gate.wait(timeout=30)  # Controlled by the local browser check.
        summary='Your trip includes a hotel stay and a return flight.'
        notification='Please review your trip bookings.'
        evidence=[{'part':part['id'],'excerpt':text[:120]}]
        if 'cancel' in text.lower():
            items=source.get('existing_items') or []
            if items:
                item=next((item for item in items if item['title']=='Hotel booking'),items[0])
                candidates=[{'id':'cancel','proposal':{'action':'cancel_item','item_id':item['id'],'expected_version':item['version'],'title':item['title']},'evidence':evidence}]
        elif 'failure-report-check' in text.lower():
            summary='Four items need review.'
            notification='An email needs review.'
            candidates=[{'id':identity,'proposal':{'action':'create_item','kind':'event' if identity!='questionnaire' else 'task','title':identity},'evidence':evidence} for identity in ['current','later','third','questionnaire']]
            candidates[0].update({'question':'How long will it take?','options':['30 minutes','1 hour']})
            if answers: candidates=candidates[2:]  # Reproduce a response that loses the answered item.
        elif 'dentist, checkup and follow-up' in text.lower():
            start=lambda day:(datetime.now(timezone.utc)+timedelta(days=day)).replace(hour=9,minute=0,second=0,microsecond=0).strftime('%Y-%m-%dT%H:%M:%SZ')
            def ask(candidate, identity):
                answer=next((a for a in answers if a.get('candidate_id')==identity),None)
                if answer:
                    candidate['proposal']['duration_minutes']=60 if answer['text']=='1 hour' else 30
                else:
                    candidate.update({'question':'How long will it take?','options':['30 minutes','45 minutes','1 hour']})
            summary='Three clinic visits, starting with the online dentist visit.'
            notification='Three clinic visits need review.'
            dentist={'id':'dentist','proposal':{'action':'create_item','kind':'event','title':'Dentist','start_at':start(6),'details':[{'label':'Meeting link','value':'https://teams.example/join/dentist'}]},'evidence':evidence}
            checkup={'id':'checkup','proposal':{'action':'create_item','kind':'event','title':'Checkup','start_at':start(7)},'evidence':evidence}
            ask(dentist,'dentist');ask(checkup,'checkup')
            candidates=[dentist,checkup,{'id':'follow-up','proposal':{'action':'create_item','kind':'event','title':'Follow-up','start_at':start(8),'duration_minutes':30},'evidence':evidence}]
        elif 'contact lens fitting' in text.lower():
            summary='A contact lens fitting needs your review.'
            notification='Please review your lens fitting.'
            start=(datetime.now(timezone.utc)+timedelta(days=5)).replace(hour=13,minute=30,second=0,microsecond=0)
            fitting={'id':'fitting','proposal':{'action':'create_item','kind':'event','title':'Contact lens fitting','start_at':start.strftime('%Y-%m-%dT%H:%M:%SZ'),'start_timezone':'Europe/Amsterdam','location':'Kinkerstraat 123, Amsterdam','phone':'020-1234567','email':'desk@example.com','website':'https://example.com','booking_reference':'AB12CD','notes':'- Bring your current glasses','reminder_offsets_minutes':[1440,60,7]},'question':'How long will it take?','options':['30 minutes','45 minutes','1 hour'],'evidence':evidence}
            if answers:
                fitting['question']='';fitting['options']=[];fitting['proposal']['duration_minutes']=45
            candidates=[fitting]
        else:
            candidates=[{'id':'hotel','proposal':{'action':'create_item','kind':'event','title':'Hotel booking','start_date':'2026-11-12','end_date':'2026-11-15','notes':'Booking ABC123'},'evidence':evidence}]
            flight={'id':'return-flight','proposal':{'action':'create_item','kind':'event','title':'Return flight'},'question':'What date is your return flight?','evidence':evidence}
            if answers:
                flight['question']='';flight['proposal'].update({'start_at':'2026-11-21T11:00:00Z','end_at':'2026-11-21T19:00:00Z','start_timezone':'Europe/Amsterdam','end_timezone':'America/New_York'})
            candidates.append(flight)
        response=json.dumps({'choices':[{'finish_reason':'stop','message':{'content':None,'tool_calls':[{'id':'capture-fixture','type':'function','function':{'name':'extract_capture','arguments':json.dumps({'summary':summary,'notification':notification,'candidates':candidates,'questions':[]})}}]}}],'usage':{'cost':0.000035}}).encode()
        self.send_response(200);self.send_header('Content-Type','application/json');self.send_header('Content-Length',str(len(response)));self.end_headers();self.wfile.write(response)
    def log_message(self,*args):pass
if __name__=='__main__':ThreadingHTTPServer(('127.0.0.1',int(os.environ.get('MODEL_FIXTURE_PORT','8181'))),Handler).serve_forever()

#!/usr/bin/env python3
"""Controlled OpenRouter boundary for local integration tests. Never deploy it.

It plays the one agent's model: chat requests, a source arrival (read the email, then
propose a review), pasted material (the same loop), source questions (find, read, answer) and
review replies (inspect the review, then update one candidate). It reads the tool results the
server sends, so every scenario exercises the real tool loop."""
import json
import os
import re
import threading
from datetime import datetime, timedelta, timezone
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

def text_of(message):
    content=message['content']
    return (content if isinstance(content,str) else json.dumps(content)).lower()

answer_gate=threading.Event()
answer_gate.set()
failed_once=set()  # sources whose first reading attempt already failed
requests=[]        # one line per scenario step the fixture played, for GET /requests
USAGE={'prompt_tokens':100,'completion_tokens':50,'cost':0.000035}

class Handler(BaseHTTPRequestHandler):
    def send_json(self,payload,status=200):
        response=json.dumps(payload).encode()
        self.send_response(status);self.send_header('Content-Type','application/json');self.send_header('Content-Length',str(len(response)));self.end_headers();self.wfile.write(response)
    def reply(self,content,name=None,args=None,calls=None):
        message={'role':'assistant','content':content}
        calls=calls or ([(name,args)] if name else [])
        if calls:message['tool_calls']=[{'id':f'fixture-call-{n}','type':'function','function':{'name':n_,'arguments':json.dumps(a)}} for n,(n_,a) in enumerate(calls)]
        self.send_json({'choices':[{'finish_reason':'stop','message':message}],'usage':USAGE})
    def do_GET(self):
        if self.path=='/requests':return self.send_json(requests)
        self.send_response(404);self.end_headers()
    def do_POST(self):
        if self.path in ('/pause-capture-answers','/resume-capture-answers'):
            answer_gate.clear() if self.path=='/pause-capture-answers' else answer_gate.set()
            self.send_response(200);self.end_headers();return
        body=json.loads(self.rfile.read(int(self.headers['Content-Length'])))
        messages=body['messages']
        # Prompt-cache breakpoints arrive as text blocks. Join them so the scripted replies read plain text.
        for message in messages:
            if isinstance(message['content'],list) and message['content'] and all(b.get('type')=='text' for b in message['content']):
                message['content']=''.join(b['text'] for b in message['content'])
        last=messages[-1]
        if last['role']=='tool':return self.after_tools(messages)
        text=text_of(last)
        if text.startswith('source arrival (data'):
            arrival=json.loads(last['content'].split('): ',1)[1].rsplit('. Start the review',1)[0])
            requests.append('arrival '+arrival['source_id'][:8])
            return self.reply(None,'read_source',{'source_id':arrival['source_id']})
        context=self.context_of(messages)
        if context and context['review']:
            requests.append('review-reply')
            answer_gate.wait(timeout=30)  # Controlled by the local browser check.
            return self.reply(None,'inspect_review',{'source_id':context['source_id']})
        if 'dentist, checkup and follow-up' in text and context:
            requests.append('paste')
            return self.reply(None,'read_source',{'source_id':context['owner_message_source_id']})
        if 'what should i bring' in text:
            requests.append('question')
            return self.reply(None,'find_sources',{'query':'checkups'})
        return self.chat(messages,body,text)
    def context_of(self,messages):
        if len(messages)<2:return None
        record=messages[-2]['content']
        if not isinstance(record,str) or '(data): ' not in record or not record.startswith(('Review context','Owner message context')):return None
        found=json.loads(record.split('(data): ',1)[1].split('}. ',1)[0]+'}')
        found['review']=record.startswith('Review context')
        return found
    def turn_message(self,messages):
        """The message that started this turn: the last user message before the tool rounds."""
        for message in reversed(messages):
            if message['role']=='user' and isinstance(message['content'],str) and not message['content'].startswith(('Images from source',)):
                return message
    def after_tools(self,messages):
        results=[]
        for message in reversed(messages):
            if message['role']!='tool':break
            results.append(message)
        names={}
        for message in reversed(messages):
            if message['role']=='assistant' and message.get('tool_calls'):
                names={c['id']:c['function']['name'] for c in message['tool_calls']};break
        result=results[-1]
        name=names.get(result['tool_call_id'])
        data=json.loads(result['content'])
        turn=self.turn_message(messages)
        turn_text=text_of(turn)
        if 'error' in data:
            return self.reply('That source is not available. I cannot answer from it.')
        if name=='find_sources':
            found=data['sources']
            if not found:return self.reply('I found no source like that.')
            return self.reply(None,'read_source',{'source_id':found[0]['id']})
        if name=='read_source':
            return self.after_read(messages,data,turn_text)
        if name=='inspect_review':
            return self.update_review(messages,data,turn)
        return self.reply('Done.')
    def after_read(self,messages,source,turn_text):
        parts=[p for p in source['parts'] if p.get('text','').strip()]
        part=parts[0] if parts else {'id':'text','text':'Booking'}
        text='\n'.join(p['text'] for p in parts)
        if 'what should i bring' in turn_text:return self.reply('Bring your insurance card and arrive ten minutes early.')
        source_id=source['source']['id']
        if 'failure-report-check' in text.lower() and source_id not in failed_once:
            failed_once.add(source_id)
            return self.send_json({'error':'fixture failure'},500)
        review=self.review(messages,text,part,source)
        revision=self.revision(messages,source)
        requests.append('review '+source_id[:8])
        args={'source_id':source_id,'expected_revision':revision,'summary':review['summary'],'notification':review['notification'],'candidates':review['candidates']}
        return self.reply(None,'propose_review',args)
    def revision(self,messages,source):
        if source['source']['kind']=='chat':return 0
        for message in messages:
            if message['role']=='user' and isinstance(message['content'],str) and message['content'].startswith('Source arrival'):
                return json.loads(message['content'].split('): ',1)[1].rsplit('. Start the review',1)[0])['revision']
        return 1
    def review(self,messages,text,part,source):
        evidence=[{'part':part['id'],'excerpt':part['text'].strip()[:120]}]
        summary,notification,candidates='Your trip includes a hotel stay and a return flight.','Please review your trip bookings.',[]
        lower=text.lower()
        if 'cancel' in lower:
            items=json.loads(messages[0]['content'].split('Current items (untrusted data): ',1)[1])
            if items:
                item=next((item for item in items if item['title']=='Hotel booking'),items[0])
                candidates=[{'proposal':{'action':'cancel_item','item_id':item['id'],'expected_version':item['version'],'title':item['title']},'evidence':evidence}]
        elif 'failure-report-check' in lower:
            summary,notification='Four items need review.','An email needs review.'
            candidates=[{'proposal':{'action':'create_item','kind':'event' if title!='questionnaire' else 'task','title':title},'evidence':evidence} for title in ['current','later','third','questionnaire']]
            candidates[0].update({'question':'How long will it take?','options':['30 minutes','1 hour']})
        elif 'dentist, checkup and follow-up' in lower:
            start=lambda day:(datetime.now(timezone.utc)+timedelta(days=day)).replace(hour=9,minute=0,second=0,microsecond=0).strftime('%Y-%m-%dT%H:%M:%SZ')
            summary,notification='Three clinic visits, starting with the online dentist visit.','Three clinic visits need review.'
            ask=lambda c:c.update({'question':'How long will it take?','options':['30 minutes','45 minutes','1 hour']})
            dentist={'proposal':{'action':'create_item','kind':'event','title':'Dentist','start_at':start(6),'details':[{'label':'Meeting link','value':'https://teams.example/join/dentist'}]},'evidence':evidence}
            checkup={'proposal':{'action':'create_item','kind':'event','title':'Checkup','start_at':start(7)},'evidence':evidence}
            ask(dentist);ask(checkup)
            candidates=[dentist,checkup,{'proposal':{'action':'create_item','kind':'event','title':'Follow-up','start_at':start(8),'duration_minutes':30},'evidence':evidence}]
        elif 'contact lens fitting' in lower:
            summary,notification='A contact lens fitting needs your review.','Please review your lens fitting.'
            start=(datetime.now(timezone.utc)+timedelta(days=5)).replace(hour=13,minute=30,second=0,microsecond=0)
            candidates=[{'proposal':{'action':'create_item','kind':'event','title':'Contact lens fitting','start_at':start.strftime('%Y-%m-%dT%H:%M:%SZ'),'start_timezone':'Europe/Amsterdam','location':'Kinkerstraat 123, Amsterdam','phone':'020-1234567','email':'desk@example.com','website':'https://example.com','booking_reference':'AB12CD','notes':'- Bring your current glasses','reminder_offsets_minutes':[1440,60,7]},'question':'How long will it take?','options':['30 minutes','45 minutes','1 hour'],'evidence':evidence}]
        else:
            candidates=[{'proposal':{'action':'create_item','kind':'event','title':'Hotel booking','start_date':'2026-11-12','end_date':'2026-11-15','notes':'Booking ABC123'},'evidence':evidence},
                        {'proposal':{'action':'create_item','kind':'event','title':'Return flight'},'question':'What date is your return flight?','evidence':evidence}]
        return {'summary':summary,'notification':notification,'candidates':candidates}
    def update_review(self,messages,review,turn):
        """The owner answered the open question: update only that candidate, from what the review holds."""
        context=json.loads(next(m['content'] for m in reversed(messages) if m['role']=='user' and isinstance(m['content'],str) and m['content'].startswith('Review context')).split('(data): ',1)[1].split('}. ',1)[0]+'}')
        answer=turn['content']
        candidate=next((c for c in review['candidates'] if c['id']==context.get('candidate_id')),None) or next(c for c in review['candidates'] if c['state']=='needs-answer')
        proposal=candidate['proposal']
        title=proposal['title']
        if title=='Return flight':
            proposal.update({'start_at':'2026-11-21T11:00:00Z','end_at':'2026-11-21T19:00:00Z','start_timezone':'Europe/Amsterdam','end_timezone':'America/New_York'})
        elif title=='Contact lens fitting':
            proposal['duration_minutes']=45
        else:
            proposal['duration_minutes']=self.minutes(answer)
        update={'id':candidate['id'],'revision':candidate['revision'],'proposal':proposal,'question':'','options':[]}
        return self.reply(None,'propose_review',{'source_id':review['source_id'],'expected_revision':review['revision'],'candidates':[update]})
    def minutes(self,answer):
        number=re.search(r'\d+',answer)
        value=int(number.group()) if number else 1
        return value*60 if 'hour' in answer.lower() else (value if number else 30)
    def chat(self,messages,body,text):
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
        self.reply(content,name,args)
    def log_message(self,*args):pass
if __name__=='__main__':ThreadingHTTPServer(('127.0.0.1',int(os.environ.get('MODEL_FIXTURE_PORT','8181'))),Handler).serve_forever()

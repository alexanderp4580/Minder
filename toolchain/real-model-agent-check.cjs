// Real-model acceptance for the one agent, on synthetic material only. Run through real-model-agent-check.
const assert=require('node:assert/strict');
const fs=require('node:fs');
(async()=>{
 const {signedHeaders}=await import('../email-worker/worker.mjs');
 const url=process.env.MINDER_URL;
 const key=Uint8Array.from(Buffer.from(process.env.MINDER_EMAIL_TEST_KEY,'base64'));
 const sleep=ms=>new Promise(r=>setTimeout(r,ms));
 const json=async(path,init)=>{const r=await fetch(url+path,init);return r.json();};
 const post=(path,body)=>fetch(url+path,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
 const until=async(what,get,ok,seconds=150)=>{let last;for(let n=0;n<seconds*2;n++){last=await get().catch(()=>null);if(last&&ok(last))return last;await sleep(500);}throw new Error('timed out: '+what+' '+JSON.stringify(last).slice(0,400));};
 const chat=()=>json('/v1/chat?after=0&limit=300');
 const replies=async(count)=>until('a chat reply',chat,h=>h.filter(e=>e.kind==='minder').length>count).then(h=>h.filter(e=>e.kind==='minder'));
 const ask=async(text)=>{const before=(await chat()).filter(e=>e.kind==='minder'||e.kind==='error').length;const r=await post('/v1/chat/messages',{text});assert.equal(r.status,202,'message accepted');
  const h=await until('reply to '+text,chat,h=>h.filter(e=>e.kind==='minder'||e.kind==='error').length>before);const last=h.filter(e=>e.kind==='minder'||e.kind==='error').pop();assert.equal(last.kind,'minder','the turn failed: '+last.text);return last.text;};
 const details='Dear patient, your dental check-up is booked at Clinic Example, Main Street 1, Utrecht on Friday 20 November 2026 at 13:00. Please bring your insurance card and arrive ten minutes early. Booking reference 0042. Fill in the attached questionnaire before your visit.';
 const questionnaire='Patient questionnaire. Required: date of birth, current medication and known allergies. Return it by 18 November 2026.';
 const raw=Buffer.from('From: Clinic <owner@example.com>\r\nSubject: Your dental check-up\r\nContent-Type: multipart/mixed; boundary=b\r\n\r\n--b\r\nContent-Type: text/plain\r\n\r\n'+details+'\r\n--b\r\nContent-Type: text/plain\r\nContent-Disposition: attachment; filename=questionnaire.txt\r\n\r\n'+questionnaire+'\r\n--b--\r\n');
 const headers=await signedHeaders({key},{from:'owner@example.com',to:'assistant@example.net',dmarc:'pass'},raw,Math.floor(Date.now()/1000),'real-1');
 const receipt=await fetch(url+'/v1/email/ingest',{method:'POST',body:raw,headers});assert.equal(receipt.status,201);
 const id=(await receipt.json()).capture_id;
 const results={};
 const review=async(captureId)=>until('a review of '+captureId,()=>json('/v1/captures/'+captureId),c=>c.candidates&&c.candidates.length>0&&['needs-answer','ready-for-review'].includes(c.status));
 const checkVisit=(c,label)=>{
  const visit=c.candidates.find(x=>/dent|check/i.test(x.proposal.title));assert.ok(visit,label+': a dental visit candidate');
  const start=new Date(visit.proposal.start_at);assert.ok(!isNaN(start),label+': the visit has a start');
  assert.equal(start.toLocaleString('en-GB',{timeZone:'Europe/Amsterdam',hour:'2-digit',minute:'2-digit',day:'2-digit',month:'2-digit'}),'20/11, 13:00',label+': the 13:00 start is kept');
  assert.ok(!visit.proposal.end_at&&!visit.proposal.duration_minutes,label+': no end or length without support: '+JSON.stringify({end:visit.proposal.end_at,minutes:visit.proposal.duration_minutes}));
  assert.ok(/long|durat|length|minutes|hour|time/i.test(visit.question||''),label+': the missing length becomes a question: '+visit.question);
  return visit;
 };
 // Forwarded email.
 let c=await review(id);const visit=checkVisit(c,'forwarded');
 results.forwarded={candidates:c.candidates.length,question:visit.question,options:visit.options};
 // The owner asks where an end time would come from: the agent checks the source and does not defend a guess.
 results.endTimeAnswer=await ask('Where does the end time of the dentist visit come from?');
 c=await json('/v1/captures/'+id);checkVisit(c,'after the evidence question');
 results.bring=await ask('What should I bring to the dentist?');assert.match(results.bring,/insurance/i,'the answer uses the email body');
 results.questionnaire=await ask('What does the questionnaire require?');assert.match(results.questionnaire,/allerg/i,'the answer uses the attachment');assert.match(results.questionnaire,/medication/i);
 const unchanged=await json('/v1/captures/'+id);assert.equal(unchanged.revision,c.revision,'questions are not answers');
 // The owner is done with the forwarded review, so the pasted copy is not a duplicate of an open one.
 c=await json('/v1/captures/'+id);r=await post('/v1/captures/'+id+'/dismiss',{revision:c.revision});assert.equal(r.status,200);
 // The same material pasted into chat gives the same review.
 const pasteText='Here is an email I received. Please propose the appointments and tasks in it for me to review.\n\n'+details+'\n\nAttachment questionnaire.txt: '+questionnaire;
 let before=(await chat()).length;r=await post('/v1/chat/messages',{text:pasteText});assert.equal(r.status,202);
 const pastedId=(await r.json()).entry.id;
 c=await review(pastedId).catch(async error=>{const tail=(await chat()).slice(-4).map(e=>e.kind+': '+(e.text||'').slice(0,300));throw new Error(error.message+' | chat tail: '+JSON.stringify(tail));});assert.equal(c.kind,'chat');const pastedVisit=checkVisit(c,'pasted');
 results.pasted={candidates:c.candidates.length,question:pastedVisit.question};
 // A targeted answer changes only its candidate.
 const others=c.candidates.filter(x=>x.id!==pastedVisit.id).map(x=>x.id+':'+x.state).sort();
 r=await post('/v1/captures/'+pastedId+'/answers',{revision:c.revision,candidate_id:pastedVisit.id,text:'45 minutes'});assert.equal(r.status,200);
 c=await until('the answered candidate',()=>json('/v1/captures/'+pastedId),x=>x.candidates.find(y=>y.id===pastedVisit.id).proposal.duration_minutes===45);
 assert.deepEqual(c.candidates.filter(x=>x.id!==pastedVisit.id).map(x=>x.id+':'+x.state).sort(),others,'only the answered candidate changed');
 results.afterAnswer={minutes:45};
 fs.writeFileSync('.local/real-model-agent-results.json',JSON.stringify(results,null,1));
 console.log(JSON.stringify({forwardedCandidates:results.forwarded.candidates,pastedCandidates:results.pasted.candidates,noUnsupportedLength:true,sourceAnswers:true,targetedAnswer:true}));
})().catch(error=>{console.error(error);process.exitCode=1;});

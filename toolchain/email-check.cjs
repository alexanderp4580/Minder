const assert=require('node:assert/strict');
const {chromium}=require(process.env.MINDER_PLAYWRIGHT||'/home/deck/SketchBook/Projects/AstraMoode/AstraCamillaGui/node_modules/playwright');
(async()=>{
 const {signedHeaders}=await import('../email-worker/worker.mjs');
 const url=process.env.MINDER_URL;
 const entriesOf=history=>{if(!Array.isArray(history))throw new Error('chat history is not a list: '+JSON.stringify(history).slice(0,300));return history;};
 const raw=Buffer.from('From: Owner <owner@example.com>\r\nSubject: Trip confirmation\r\nContent-Type: multipart/mixed; boundary=trip\r\n\r\n--trip\r\nContent-Type: text/plain\r\n\r\nHotel booking ABC123 and return flight.\r\n--trip\r\nContent-Type: text/calendar\r\nContent-Disposition: attachment; filename=hotel.ics\r\n\r\nBEGIN:VCALENDAR\r\nVERSION:2.0\r\nBEGIN:VEVENT\r\nUID:hotel-abc123\r\nDTSTART;VALUE=DATE:20261112\r\nDTEND;VALUE=DATE:20261115\r\nSUMMARY:Hotel booking\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n--trip--\r\n');
 const sign=async(bytes,id)=>signedHeaders({key:Uint8Array.from(Buffer.from(process.env.MINDER_EMAIL_TEST_KEY,'base64'))},{from:'owner@example.com',to:'assistant@example.net',dmarc:'pass'},bytes,Math.floor(Date.now()/1000),id);
 const headers=await sign(raw,'first');
 let response=await fetch(url+'/v1/email/ingest',{method:'POST',body:raw,headers});assert.equal(response.status,201);const id=(await response.json()).capture_id;
 response=await fetch(url+'/v1/email/ingest',{method:'POST',body:raw,headers:await sign(raw,'retry')});assert.equal(response.status,200);assert.equal((await response.json()).capture_id,id);
 let source;
 for(let n=0;n<100;n++){source=await fetch(url+'/v1/captures/'+id).then(r=>r.json());if(source.candidates.length===2)break;await new Promise(resolve=>setTimeout(resolve,100));}
 assert.deepEqual(source.candidates.map(c=>c.state),['pending','queued'],'only the first appointment is published');
 assert.deepEqual(await fetch(url+'/v1/captures/'+id+'/attachments/original').then(r=>r.arrayBuffer()).then(b=>Buffer.from(b)),raw);
 assert.equal((await fetch(url+'/v1/items?from=2026-11-01T00:00:00Z&to=2026-12-01T00:00:00Z').then(r=>r.json())).length,0);
 const browser=await chromium.launch({executablePath:process.env.MINDER_CHROMIUM||'/home/deck/.cache/ms-playwright/chromium-1243/chrome-linux64/chrome',args:['--no-sandbox']});
 let page;
 try{
  page=await browser.newPage({viewport:{width:412,height:892}});const errors=[];page.on('pageerror',error=>errors.push(error.message));
  await page.route('**/v1/events?*',route=>route.abort());
  await page.goto(url);await page.waitForSelector('flutter-view',{timeout:60000});await page.locator('flt-semantics-placeholder').evaluate(element=>element.click()).catch(()=>{});
  // The hotel comes first: confirm it in chat, then the return flight asks its question.
  await page.getByRole('button',{name:'Confirm',exact:true}).last().waitFor({timeout:30000});await page.getByRole('button',{name:'Confirm',exact:true}).last().click();
  await page.getByRole('group',{name:/PREVIEW.*Hotel booking.*Confirmed/}).waitFor({timeout:10000});
  assert.equal(await page.getByRole('button',{name:'Confirm',exact:true}).count(),0,'the original Confirm is replaced even with the event stream blocked');
  await page.unroute('**/v1/events?*');
  const answer=page.getByRole('textbox',{name:'Reply'});
  await answer.waitFor({timeout:30000});await answer.fill('21 November');await page.getByRole('button',{name:'Send answer',exact:true}).click();
  for(let attempt=0;attempt<100;attempt++){const current=await fetch(url+'/v1/captures/'+id).then(r=>r.json());if(current.status==='ready-for-review')break;await new Promise(resolve=>setTimeout(resolve,100));}
  const reviewed=await fetch(url+'/v1/captures/'+id).then(r=>r.json());assert.equal(reviewed.status,'ready-for-review');assert.deepEqual(reviewed.candidates.map(c=>c.state),['confirmed','pending']);
  // Reload the completed revision so newly arriving previews cannot move the scroll target.
  await page.reload();await page.waitForSelector('flutter-view',{timeout:60000});await page.locator('flt-semantics-placeholder').evaluate(element=>element.click()).catch(()=>{});
  await page.getByRole('button',{name:'Confirm',exact:true}).last().waitFor({timeout:30000});
  const original=page.getByRole('button',{name:'original.eml',exact:true});
  // The chat is long now: scroll up until the email card is on screen.
  await page.mouse.move(200,450);
  const onScreen=async()=>{const box=await original.first().boundingBox({timeout:1000}).catch(()=>null);return !!box&&box.y>=60&&box.y+box.height<=840;};
  for(let n=0;n<40&&!(await onScreen());n++){await page.mouse.wheel(0,-400);await page.waitForTimeout(500);}
  await original.waitFor();const downloadPromise=page.waitForEvent('download');downloadPromise.catch(()=>{});await original.click();const download=await downloadPromise;assert.equal(download.suggestedFilename(),'original.eml');await page.mouse.wheel(0,20000);await page.waitForTimeout(300);
  await page.getByRole('button',{name:'Confirm',exact:true}).last().click();
  for(let attempt=0;attempt<100;attempt++){const current=await fetch(url+'/v1/captures/'+id).then(r=>r.json());if(current.candidates.every(c=>c.state==='confirmed'))break;await new Promise(resolve=>setTimeout(resolve,100));}
  const tripDone=await fetch(url+'/v1/captures/'+id).then(r=>r.json());assert.ok(tripDone.candidates.every(c=>c.state==='confirmed'),'both trip items are confirmed');
  const tripChat=entriesOf(await fetch(url+'/v1/chat?after=0&limit=200').then(r=>r.json())).filter(e=>e.kind==='minder').map(e=>e.text);
  assert.ok(tripChat.includes('Your trip includes a hotel stay and a return flight.'),'the opening message counts the appointments: '+JSON.stringify(tripChat));
  assert.ok(tripChat.includes('Appointment 2 of 2: Return flight'),'progress is announced');assert.ok(tripChat.includes('Done: 2 added, 0 skipped.'),'the review ends with a summary');
  await page.getByText('Calendar',{exact:true}).click();await page.waitForTimeout(500);
  await page.screenshot({path:'.local/email-calendar.png'});
  const items=await fetch(url+'/v1/items?from=2026-11-01T00:00:00Z&to=2026-12-01T00:00:00Z&timezone=Europe%2FAmsterdam').then(r=>r.json());assert.equal(items.length,2);assert.equal(items.find(i=>i.title==='Return flight').end_local,'2026-11-21T14:00');
  const cancellation=Buffer.from('From: Owner <owner@example.com>\r\nSubject: Booking cancellation\r\n\r\nCancel hotel booking ABC123.');
  const cancelledReceipt=await fetch(url+'/v1/email/ingest',{method:'POST',body:cancellation,headers:await sign(cancellation,'cancellation')});assert.equal(cancelledReceipt.status,201);const cancellationId=(await cancelledReceipt.json()).capture_id;
  let cancellationReview;
  for(let n=0;n<100;n++){cancellationReview=await fetch(url+'/v1/captures/'+cancellationId).then(r=>r.json());if(cancellationReview.status==='ready-for-review')break;await new Promise(resolve=>setTimeout(resolve,100));}
  assert.equal(cancellationReview.status,'ready-for-review');
  const cancelProposal=cancellationReview.candidates.find(candidate=>candidate.state==='pending');assert.equal(cancelProposal.proposal.action,'cancel_item');
  const beforeCancellation=await fetch(url+'/v1/items?from=2026-11-01T00:00:00Z&to=2026-12-01T00:00:00Z').then(r=>r.json());assert.equal(beforeCancellation.find(item=>item.title==='Hotel booking').status,'open');
  assert.equal((await fetch(url+'/v1/changes/'+cancelProposal.change_id+'/confirm',{method:'POST'})).status,200);
  const afterCancellation=await fetch(url+'/v1/items?from=2026-11-01T00:00:00Z&to=2026-12-01T00:00:00Z').then(r=>r.json());assert.equal(afterCancellation.find(item=>item.title==='Hotel booking').status,'cancelled');assert.equal(afterCancellation.find(item=>item.title==='Return flight').status,'open');
  // Appointment flow: received card, a separate question message with quick replies, then one complete item.
  const fitting=Buffer.from('From: Owner <owner@example.com>\r\nSubject: Contact lens fitting\r\n\r\nContact lens fitting at Specsavers. Booking reference AB12CD. Phone 020-1234567. Email desk@example.com. Website https://example.com.');
  const fittingReceipt=await fetch(url+'/v1/email/ingest',{method:'POST',body:fitting,headers:await sign(fitting,'fitting')});assert.equal(fittingReceipt.status,201);const fittingId=(await fittingReceipt.json()).capture_id;
  let questionEntry;
  for(let n=0;n<100;n++){const history=await fetch(url+'/v1/chat?after=0&limit=100').then(r=>r.json());questionEntry=entriesOf(history).find(e=>e.kind==='question'&&e.capture_id===fittingId);if(questionEntry)break;await new Promise(resolve=>setTimeout(resolve,100));}
  assert.ok(questionEntry,'the question must arrive as its own entry');assert.equal(questionEntry.capture.candidates[0].question,'How long will it take?');assert.deepEqual(questionEntry.capture.candidates[0].options,['30 minutes','45 minutes','1 hour']);
  await page.reload();await page.waitForSelector('flutter-view',{timeout:60000});await page.locator('flt-semantics-placeholder').evaluate(element=>element.click()).catch(()=>{});
  const fittingDetails=page.getByRole('button',{name:/Contact lens fitting/});
  await page.mouse.move(200,450);
  for(let n=0;n<30;n++){const box=await fittingDetails.boundingBox({timeout:1000}).catch(()=>null);if(box&&box.y>=60&&box.y+box.height<=800)break;await page.mouse.wheel(0,-300);await page.waitForTimeout(150);}
  await fittingDetails.click();
  await page.getByRole('button',{name:'30 minutes',exact:true}).nth(1).waitFor();
  const expandedEmail=await page.locator('body').ariaSnapshot();
  assert.match(expandedEmail,/Booking reference/,'expanded email labels the reference');
  assert.match(expandedEmail,/AB12CD/,'expanded email exposes the selectable booking reference');
  assert.match(expandedEmail,/020-1234567/,'expanded email shows the phone');
  assert.doesNotMatch(expandedEmail,/1_1:/,'raw evidence IDs stay out of the card');
  await page.screenshot({path:'.local/email-expanded-details.png'});
  const quick=page.getByRole('button',{name:'45 minutes',exact:true}).first();await quick.waitFor({timeout:30000});
  const fixture=process.env.OPENROUTER_URL;
  assert.equal((await fetch(fixture+'/pause-capture-answers',{method:'POST'})).status,200);
  try {
   await quick.click();
   await page.waitForTimeout(500);
   await page.screenshot({path:'.local/email-held.png'});
   require('node:fs').writeFileSync('.local/email-held-aria.txt',await page.locator('body').ariaSnapshot());
   await page.getByText('Working',{exact:true}).waitFor({state:'visible',timeout:10000});
   await page.screenshot({path:'.local/email-working.png'});
  } finally { await fetch(fixture+'/resume-capture-answers',{method:'POST'}); }
  let fittingReview;
  for(let n=0;n<100;n++){fittingReview=await fetch(url+'/v1/captures/'+fittingId).then(r=>r.json());if(fittingReview.status==='ready-for-review')break;await new Promise(resolve=>setTimeout(resolve,100));}
  assert.equal(fittingReview.status,'ready-for-review');
  const chip=page.getByRole('button',{name:'2 hours',exact:true});await chip.waitFor({timeout:30000});await chip.click();
  for(let n=0;n<100;n++){const history=await fetch(url+'/v1/chat?after=0&limit=100').then(r=>r.json());const preview=entriesOf(history).find(e=>e.kind==='preview'&&e.capture_id===fittingId);if(preview&&preview.change.reminder_offsets_minutes.includes(120))break;await new Promise(resolve=>setTimeout(resolve,100));}
  await page.getByRole('button',{name:'Confirm',exact:true}).last().click();
  let saved;
  for(let n=0;n<100;n++){const all=await fetch(url+'/v1/items?from=2026-11-01T00:00:00Z&to=2026-12-01T00:00:00Z').then(r=>r.json());saved=all.find(i=>i.title==='Contact lens fitting');if(saved)break;await new Promise(resolve=>setTimeout(resolve,100));}
  assert.ok(saved,'the appointment must be saved');assert.equal(saved.location,'Kinkerstraat 123, Amsterdam');assert.equal(saved.duration_minutes,45);assert.equal(saved.notes,'- Bring your current glasses');assert.equal(saved.phone,'020-1234567');assert.equal(saved.email,'desk@example.com');assert.equal(saved.website,'https://example.com');assert.equal(saved.booking_reference,'AB12CD');assert.deepEqual(saved.reminder_offsets_minutes,[1440,120,60]);
  await page.screenshot({path:'.local/email-appointment.png'});
  // Three appointments, one a Teams meeting: reviewed one at a time, durations answered by the model.
  const checkups=Buffer.from('From: Owner <owner@example.com>\r\nSubject: Checkups\r\n\r\nDentist, checkup and follow-up appointments at the clinic. The dentist visit is a Microsoft Teams meeting.');
  const checkupsReceipt=await fetch(url+'/v1/email/ingest',{method:'POST',body:checkups,headers:await sign(checkups,'checkups')});assert.equal(checkupsReceipt.status,201);const checkupsId=(await checkupsReceipt.json()).capture_id;
  const capture=async(predicate)=>{let c;for(let n=0;n<100;n++){c=await fetch(url+'/v1/captures/'+checkupsId).then(r=>r.json());if(predicate(c))return c;await new Promise(resolve=>setTimeout(resolve,100));}throw new Error('capture never reached the expected state: '+JSON.stringify(c.candidates.map(x=>x.state)));};
  const states=c=>c.candidates.map(x=>x.state).join(' ');
  let checkupsCapture=await capture(c=>c.candidates.length===3);assert.equal(states(checkupsCapture),'needs-answer queued queued');
  assert.equal(checkupsCapture.candidates[0].proposal.location||'','','a Teams meeting has no address');
  const chatTexts=async()=>entriesOf(await fetch(url+'/v1/chat?after=0&limit=300').then(r=>r.json())).filter(e=>e.kind==='minder').map(e=>e.text);
  let texts=await chatTexts();assert.ok(texts.includes('Three clinic visits, starting with the online dentist visit.'),JSON.stringify(texts));assert.ok(texts.includes('Appointment 1 of 3: Dentist'));
  const answerTo=async(captureId,title,text)=>{const c=await fetch(url+'/v1/captures/'+captureId).then(r=>r.json());const candidate=c.candidates.find(x=>x.proposal.title===title);assert.ok(candidate,'candidate '+title);const r=await fetch(url+'/v1/captures/'+captureId+'/answers',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({revision:c.revision,candidate_id:candidate.id,text})});assert.equal(r.status,200,'answer '+title);const entry=await r.json();assert.equal(entry.kind,'user','a quick reply is the owner\'s own message');assert.equal(entry.text,text);};
  await answerTo(checkupsId,'Dentist','1 hour');checkupsCapture=await capture(c=>c.candidates[0].state==='pending');assert.equal(checkupsCapture.candidates[0].proposal.duration_minutes,60,'the model used the duration answer');assert.equal(states(checkupsCapture),'pending queued queued');assert.deepEqual(checkupsCapture.candidates[0].proposal.details,[{label:'Meeting link',value:'https://teams.example/join/dentist'}]);
  assert.equal((await fetch(url+'/v1/changes/'+checkupsCapture.candidates[0].change_id+'/confirm',{method:'POST'})).status,200);
  checkupsCapture=await capture(c=>c.candidates[1].state==='needs-answer');await answerTo(checkupsId,'Checkup','30 minutes');checkupsCapture=await capture(c=>c.candidates[1].state==='pending');
  assert.equal((await fetch(url+'/v1/changes/'+checkupsCapture.candidates[1].change_id+'/discard',{method:'POST'})).status,200);
  checkupsCapture=await capture(c=>c.candidates[2].state==='pending');assert.equal((await fetch(url+'/v1/changes/'+checkupsCapture.candidates[2].change_id+'/confirm',{method:'POST'})).status,200);
  texts=await chatTexts();assert.equal(texts[texts.length-1],'Done: 2 added, 1 skipped.','the review ends with a summary');
  const checkupItems=(await fetch(url+'/v1/items?from=2026-11-01T00:00:00Z&to=2026-12-01T00:00:00Z').then(r=>r.json())).filter(i=>['Dentist','Checkup','Follow-up'].includes(i.title));assert.deepEqual(checkupItems.map(i=>i.title).sort(),['Dentist','Follow-up']);assert.deepEqual(checkupItems.find(i=>i.title==='Dentist').details,[{label:'Meeting link',value:'https://teams.example/join/dentist'}]);
  // A model failure while reading is reported in chat, keeps the original and recovers with Retry.
  const failureRaw=Buffer.from('From: Owner <owner@example.com>\r\nSubject: Failure report check\r\n\r\nfailure-report-check: three appointments and a questionnaire.');
  const failureReceipt=await fetch(url+'/v1/email/ingest',{method:'POST',body:failureRaw,headers:await sign(failureRaw,'failure-report')});assert.equal(failureReceipt.status,201);const failureId=(await failureReceipt.json()).capture_id;
  let failed;
  for(let n=0;n<100;n++){failed=await fetch(url+'/v1/captures/'+failureId).then(r=>r.json());if(failed.status==='failed')break;await new Promise(r=>setTimeout(r,100));}
  assert.equal(failed.status,'failed');assert.equal(failed.candidates.length,0);assert.match(failed.error,/something went wrong/);
  const failedHistory=entriesOf(await fetch(url+'/v1/chat?after=0&limit=300').then(r=>r.json()));
  assert.equal(failedHistory[failedHistory.length-1].kind,'minder');assert.equal(failedHistory[failedHistory.length-1].text,failed.error);
  await page.reload();await page.waitForSelector('flutter-view',{timeout:60000});await page.locator('flt-semantics-placeholder').evaluate(e=>e.click()).catch(()=>{});
  await page.getByText(failed.error,{exact:false}).last().waitFor({timeout:10000});
  assert.equal(await page.getByText('Working',{exact:true}).count(),0,'a failed turn ends Working');
  assert.equal((await fetch(url+'/v1/captures/'+failureId+'/retry',{method:'POST'})).status,200);
  for(let n=0;n<100;n++){failed=await fetch(url+'/v1/captures/'+failureId).then(r=>r.json());if(failed.status==='needs-answer')break;await new Promise(r=>setTimeout(r,100));}
  assert.equal(failed.status,'needs-answer','Retry reads the email again');assert.equal(failed.candidates.length,4);
  // Pasted material: a long paste (over 4,000 characters) goes through the same agent and gives the same review.
  const filler='Parking is behind the clinic and opening hours are posted at the door.\n'.repeat(80);
  const pasted='Dentist, checkup and follow-up appointments at the clinic. The dentist visit is a Microsoft Teams meeting.\n'+filler;
  assert.ok(pasted.length>4000);
  const accepted=await fetch(url+'/v1/chat/messages',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({text:pasted})});assert.equal(accepted.status,202);
  const pastedId=(await accepted.json()).entry.id;
  const pastedCapture=async(predicate)=>{let c;for(let n=0;n<150;n++){c=await fetch(url+'/v1/captures/'+pastedId).then(r=>r.ok?r.json():null);if(c&&predicate(c))return c;await new Promise(resolve=>setTimeout(resolve,100));}throw new Error('pasted review never reached the expected state: '+JSON.stringify(c));};
  let pastedReview=await pastedCapture(c=>c.candidates.length===3);
  assert.equal(pastedReview.kind,'chat');assert.equal(states(pastedReview),'needs-answer queued queued','the pasted review matches the forwarded one');
  assert.deepEqual(pastedReview.candidates.map(c=>c.proposal.title),['Dentist','Checkup','Follow-up']);
  // A question during the open review is answered from the source and changes nothing.
  const asked=await fetch(url+'/v1/chat/messages',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({text:'What should I bring to the dentist?'})});assert.equal(asked.status,202);
  for(let n=0;n<100;n++){const texts2=await chatTexts();if(texts2.includes('Bring your insurance card and arrive ten minutes early.'))break;await new Promise(resolve=>setTimeout(resolve,100));}
  assert.ok((await chatTexts()).includes('Bring your insurance card and arrive ten minutes early.'),'the answer comes from the retained source');
  const afterQuestion=await pastedCapture(()=>true);assert.equal(states(afterQuestion),'needs-answer queued queued');assert.equal(afterQuestion.revision,pastedReview.revision,'a question is not an answer');
  // A targeted answer changes only its candidate.
  await answerTo(pastedId,'Dentist','1 hour');
  pastedReview=await pastedCapture(c=>c.candidates[0].state==='pending');
  assert.equal(pastedReview.candidates[0].proposal.duration_minutes,60);assert.equal(states(pastedReview),'pending queued queued');
  assert.equal(pastedReview.candidates[1].question,'How long will it take?');
  // The agent works through the tool loop, reading the source before it proposes.
  const played=await fetch(process.env.OPENROUTER_URL+'/requests').then(r=>r.json());
  assert.ok(played.some(line=>line.startsWith('paste')),'the paste was read by the agent');assert.ok(played.some(line=>line.startsWith('arrival')),'an arrival was read by the agent');
  // Deleting the pasted source needs a confirmation and redacts its text.
  const deletion=await fetch(url+'/v1/captures/'+pastedId+'/delete',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({revision:pastedReview.revision})}).then(r=>r.json());
  assert.equal((await fetch(url+'/v1/changes/'+deletion.change.id+'/confirm',{method:'POST'})).status,200);
  assert.equal((await fetch(url+'/v1/captures/'+pastedId).then(r=>r.json())).deleted,true);
  assert.ok(!entriesOf(await fetch(url+'/v1/chat?after=0&limit=400').then(r=>r.json())).some(e=>e.kind==='user'&&e.text.includes('Parking is behind the clinic')),'the pasted text is gone from the chat');
  assert.deepEqual(errors,[]);
  console.log('PASS: compiled web email review, targeted answer, owner download, confirmation, travel calendar, reviewed cancellation, a three-appointment email reviewed one at a time, a visible failure with Retry, and the same review from a long pasted text with a source question and deletion.');
 }catch(error){if(page){require('node:fs').writeFileSync('.local/email-accessibility.txt',await page.locator('body').ariaSnapshot().catch(()=>''));require('node:fs').writeFileSync('.local/email-dom.html',await page.content().catch(()=>''));}if(page)await page.screenshot({path:'.local/email-failure.png'}).catch(()=>{});throw error;}finally{await browser.close();}
})().catch(error=>{console.error(error);process.exitCode=1;});

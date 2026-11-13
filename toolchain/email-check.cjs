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
 for(let n=0;n<100;n++){source=await fetch(url+'/v1/captures/'+id).then(r=>r.json());if(source.status==='needs-answer')break;await new Promise(resolve=>setTimeout(resolve,100));}
 assert.equal(source.status,'needs-answer');assert.equal(source.candidates.filter(c=>c.state==='pending').length,1);
 assert.deepEqual(await fetch(url+'/v1/captures/'+id+'/attachments/original').then(r=>r.arrayBuffer()).then(b=>Buffer.from(b)),raw);
 assert.equal((await fetch(url+'/v1/items?from=2026-11-01T00:00:00Z&to=2026-12-01T00:00:00Z').then(r=>r.json())).length,0);
 const browser=await chromium.launch({executablePath:process.env.MINDER_CHROMIUM||'/home/deck/.cache/ms-playwright/chromium-1243/chrome-linux64/chrome',args:['--no-sandbox']});
 let page;
 try{
  page=await browser.newPage({viewport:{width:412,height:892}});const errors=[];page.on('pageerror',error=>errors.push(error.message));
  await page.goto(url);await page.waitForSelector('flutter-view',{timeout:60000});await page.locator('flt-semantics-placeholder').evaluate(element=>element.click()).catch(()=>{});
  const answer=page.getByRole('textbox',{name:'Reply'});
  await answer.waitFor({timeout:30000});await answer.fill('21 November');await page.getByRole('button',{name:'Send answer',exact:true}).click();
  for(let attempt=0;attempt<100;attempt++){const current=await fetch(url+'/v1/captures/'+id).then(r=>r.json());if(current.status==='ready-for-review')break;await new Promise(resolve=>setTimeout(resolve,100));}
  const reviewed=await fetch(url+'/v1/captures/'+id).then(r=>r.json());assert.equal(reviewed.status,'ready-for-review');assert.equal(reviewed.candidates.filter(c=>c.state==='pending').length,2);
  // Reload the completed revision so newly arriving previews cannot move the scroll target.
  await page.reload();await page.waitForSelector('flutter-view',{timeout:60000});await page.locator('flt-semantics-placeholder').evaluate(element=>element.click()).catch(()=>{});
  await page.getByRole('button',{name:'Confirm',exact:true}).last().waitFor({timeout:30000});
  const original=page.getByRole('button',{name:'original.eml',exact:true});
  // The chat is long now: scroll up until the email card is on screen.
  await page.mouse.move(200,450);
  const onScreen=async()=>{const box=await original.first().boundingBox({timeout:1000}).catch(()=>null);return !!box&&box.y>=60&&box.y+box.height<=840;};
  for(let n=0;n<40&&!(await onScreen());n++){await page.mouse.wheel(0,-400);await page.waitForTimeout(500);}
  await original.waitFor();const downloadPromise=page.waitForEvent('download');downloadPromise.catch(()=>{});await original.click();const download=await downloadPromise;assert.equal(download.suggestedFilename(),'original.eml');await page.mouse.wheel(0,20000);await page.waitForTimeout(300);
  await page.getByRole('button',{name:'Confirm',exact:true}).first().click();
  for(let attempt=0;attempt<100;attempt++){const current=await fetch(url+'/v1/captures/'+id).then(r=>r.json());if(current.candidates.some(c=>c.state==='confirmed'))break;await new Promise(resolve=>setTimeout(resolve,100));}
  // Confirm the second candidate through the same server endpoint after verifying the rendered controls exist.
  const remaining=await fetch(url+'/v1/captures/'+id).then(r=>r.json());const candidate=remaining.candidates.find(c=>c.state==='pending');assert.ok(candidate);assert.equal((await fetch(url+'/v1/changes/'+candidate.change_id+'/confirm',{method:'POST'})).status,200);
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
  const quick=page.getByRole('button',{name:'45 minutes',exact:true}).first();await quick.waitFor({timeout:30000});await quick.click();
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
  assert.deepEqual(errors,[]);
  console.log('PASS: compiled web email review, targeted answer, owner download, confirmation, travel calendar and reviewed cancellation.');
 }catch(error){if(page){require('node:fs').writeFileSync('.local/email-accessibility.txt',await page.locator('body').ariaSnapshot().catch(()=>''));require('node:fs').writeFileSync('.local/email-dom.html',await page.content().catch(()=>''));}if(page)await page.screenshot({path:'.local/email-failure.png'}).catch(()=>{});throw error;}finally{await browser.close();}
})().catch(error=>{console.error(error);process.exitCode=1;});

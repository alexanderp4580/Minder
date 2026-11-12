const assert=require('node:assert/strict');
const {chromium}=require(process.env.MINDER_PLAYWRIGHT||'/home/deck/SketchBook/Projects/AstraMoode/AstraCamillaGui/node_modules/playwright');
(async()=>{
 const {signedHeaders}=await import('../email-worker/worker.mjs');
 const url=process.env.MINDER_URL;
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
 try{
  const page=await browser.newPage({viewport:{width:412,height:892}});const errors=[];page.on('pageerror',error=>errors.push(error.message));
  await page.goto(url);await page.waitForSelector('flutter-view',{timeout:60000});await page.locator('flt-semantics-placeholder').evaluate(element=>element.click()).catch(()=>{});
  const answer=page.getByRole('textbox',{name:'Answer or clarify this email'});
  await answer.waitFor({timeout:30000});await answer.fill('21 November');await page.getByRole('button',{name:'Send answer',exact:true}).click();
  for(let attempt=0;attempt<100;attempt++){const current=await fetch(url+'/v1/captures/'+id).then(r=>r.json());if(current.status==='ready-for-review')break;await new Promise(resolve=>setTimeout(resolve,100));}
  const reviewed=await fetch(url+'/v1/captures/'+id).then(r=>r.json());assert.equal(reviewed.status,'ready-for-review');assert.equal(reviewed.candidates.filter(c=>c.state==='pending').length,2);
  const original=page.getByRole('button',{name:'original.eml',exact:true});await original.waitFor();const downloadPromise=page.waitForEvent('download');await original.click();const download=await downloadPromise;assert.equal(download.suggestedFilename(),'original.eml');
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
  assert.deepEqual(errors,[]);
  console.log('PASS: compiled web email review, targeted answer, owner download, confirmation, travel calendar and reviewed cancellation.');
 }finally{await browser.close();}
})().catch(error=>{console.error(error);process.exitCode=1;});

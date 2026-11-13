// Run with Node and MINDER_PLAYWRIGHT pointing to an installed Playwright package.
const {chromium}=require(process.env.MINDER_PLAYWRIGHT||'/home/deck/SketchBook/Projects/AstraMoode/AstraCamillaGui/node_modules/playwright');
const assert=require('node:assert/strict');
(async()=>{
 const browser=await chromium.launch({executablePath:process.env.MINDER_CHROMIUM||'/home/deck/.cache/ms-playwright/chromium-1243/chrome-linux64/chrome',headless:true,args:['--no-sandbox']});
 const page=await browser.newPage({viewport:{width:412,height:892}});const errors=[];page.on('pageerror',e=>errors.push(e.message));
 const url=process.env.MINDER_URL||'http://127.0.0.1:8180';
 async function semantics(){await page.waitForSelector('flutter-view',{timeout:60000});await page.locator('flt-semantics-placeholder').evaluate(e=>e.click()).catch(()=>{});await page.getByRole('textbox').waitFor({timeout:30000});}
 async function send(text){console.log('Send:',text);await page.getByRole('textbox').click();await page.keyboard.type(text,{delay:20});await page.waitForTimeout(300);await page.getByRole('button',{name:'Send',exact:true}).click();await page.waitForTimeout(300);}
 async function button(name){await page.getByRole('button',{name,exact:true}).last().click();}
 async function has(text){const escaped=text.replace(/[.*+?^${}()|[\]\\]/g,'\\$&');await page.getByText(text,{exact:false}).or(page.getByRole('group',{name:new RegExp(escaped)})).last().waitFor({timeout:30000});}
 page.on('response',r=>{if(r.url().includes('/v1/')&&r.request().method()==='POST')console.log('API',r.status(),new URL(r.url()).pathname)});await page.goto(url);await semantics();await page.screenshot({path:'docs/evidence/local-empty.png'});
 await send('Call the plumber on Thursday at 10.');await page.getByRole('button',{name:'Confirm',exact:true}).waitFor();
 let count=await page.request.get(url+'/v1/chat').then(r=>r.json()).then(es=>es.filter(e=>e.kind==='result').length);assert.equal(count,0,'Preview wrote a result');
 await page.screenshot({path:'docs/evidence/local-preview.png'});await button('Confirm');await has('Saved: Call plumber.');
 await send('What is open?');await has('Open items:');
 await send('Move the plumber to Friday.');await page.getByRole('button',{name:'Discard',exact:true}).waitFor();await button('Discard');await has('Discarded');
 await send('Move the plumber to Friday.');await page.getByRole('button',{name:'Confirm',exact:true}).waitFor();await button('Confirm');await has('Saved: Call plumber.');
 await send('Mark the plumber done.');await page.getByRole('button',{name:'Confirm',exact:true}).waitFor();await button('Confirm');
 await page.waitForFunction(async()=>{const es=await fetch('/v1/chat').then(r=>r.json());return es.some(e=>e.kind==='result'&&e.change?.action==='complete_item');});
 await page.getByRole('button',{name:'Confirm',exact:true}).waitFor({state:'detached'});await page.waitForTimeout(300);await button('Undo');await has('Undone: Call plumber.');await page.reload();await semantics();await has('Undone: Call plumber.');
 await page.context().setOffline(true);await send('What is open?');
 const offline=page.getByRole('button',{name:'Offline. Tap to retry',exact:true});await offline.waitFor({timeout:10000});
 assert.equal(await page.getByText('Check your connection.',{exact:false}).count(),0,'Offline transport failures must not add a message');
 assert.equal(await page.getByRole('button',{name:'Retry',exact:true}).count(),0,'Offline recovery belongs in the header');
 await page.getByRole('textbox').click();
 const restored=await page.getByRole('textbox').evaluate(e=>({value:e.value,text:e.textContent,aria:e.getAttribute('aria-valuetext'),tag:e.tagName}));
 assert.equal(restored.value??restored.aria??restored.text,'What is open?',`A failed send must return its text to the composer (${restored.tag})`);
 await page.context().setOffline(false);await offline.click();await offline.waitFor({state:'detached'});
 await button('Send');await has('Open items:');
 await page.screenshot({path:'docs/evidence/local-flow.png'});await page.setViewportSize({width:1280,height:900});await page.screenshot({path:'docs/evidence/local-desktop.png'});
 assert.deepEqual(errors,[]);console.log('PASS: compiled Flutter web against local Go/SQLite: preview, confirm, query, discard, change, complete, undo, reload, offline icon/recovery and restored send; phone and desktop layouts.');
 await browser.close();
})().catch(e=>{console.error(e);process.exit(1);});

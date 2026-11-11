// Run with Node and MINDER_PLAYWRIGHT pointing to an installed Playwright package.
const {chromium}=require(process.env.MINDER_PLAYWRIGHT||'/home/deck/SketchBook/Projects/AstraMoode/AstraCamillaGui/node_modules/playwright');
const assert=require('node:assert/strict');
(async()=>{
 const browser=await chromium.launch({executablePath:process.env.MINDER_CHROMIUM||'/home/deck/.cache/ms-playwright/chromium-1243/chrome-linux64/chrome',headless:true,args:['--no-sandbox']});
 const page=await browser.newPage({viewport:{width:412,height:892}});const errors=[];page.on('pageerror',e=>errors.push(e.message));
 const url=process.env.MINDER_URL||'http://127.0.0.1:8180';
 async function semantics(){await page.waitForSelector('flutter-view',{timeout:60000});await page.locator('flt-semantics-placeholder').evaluate(e=>e.click()).catch(()=>{});await page.getByRole('textbox').waitFor({timeout:30000});}
 async function button(name){await page.getByRole('button',{name,exact:true}).last().click();}
 await page.goto(url);await semantics();
 await page.getByRole('textbox').click();await page.keyboard.type('Remind me to water the plants.',{delay:20});await page.waitForTimeout(300);await page.getByRole('button',{name:'Send',exact:true}).click();
 await page.getByRole('button',{name:'Confirm',exact:true}).waitFor();
 let entries=await page.request.get(url+'/v1/chat').then(r=>r.json());assert.equal(entries.filter(e=>e.reminder).length,0,'Reminder before Confirm');
 await button('Confirm');
 await page.waitForFunction(async()=>{const es=await fetch('/v1/chat').then(r=>r.json());return es.some(e=>e.reminder&&e.reminder.state==='active');},null,{timeout:45000});await page.getByRole('button',{name:'Done',exact:true}).last().waitFor({timeout:15000});
 
 await page.screenshot({path:'docs/evidence/local-reminder.png'});for(const name of ['Done','Open'])await page.getByRole('button',{name,exact:true}).last().waitFor();
 await button('Done');
 await page.waitForFunction(async()=>{const es=await fetch('/v1/chat').then(r=>r.json());return es.some(e=>e.reminder&&e.reminder.state!=='active');},null,{timeout:30000});
 assert.deepEqual(errors,[]);console.log('PASS: due reminder appeared in chat with Done, Snooze and Open; Done handled it.');
 await browser.close();
})().catch(e=>{console.error(e);process.exit(1);});

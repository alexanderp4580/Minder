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
 await page.getByRole('textbox').click();await page.keyboard.type('Call the plumber on Thursday at 10.',{delay:20});await page.waitForTimeout(300);await page.getByRole('button',{name:'Send',exact:true}).click();
 await page.getByRole('button',{name:'Confirm',exact:true}).waitFor();await button('Confirm');
 await page.waitForFunction(async()=>{const es=await fetch('/v1/chat').then(r=>r.json());return es.some(e=>e.kind==='result');},null,{timeout:30000});
 await button('Calendar');
 await page.getByRole('button',{name:/NOV 12, 1 item/}).waitFor({timeout:30000});
 await page.getByRole('button',{name:/NOV 12, 1 item/}).click();
 await page.getByRole('group',{name:/Call plumber/}).waitFor({timeout:10000});
 await button('Details');await page.getByRole('group',{name:/Call plumber[\s\S]*Type[\s\S]*Task/}).waitFor();
 await button('Hide details');await page.getByRole('group',{name:/Call plumber[\s\S]*Type[\s\S]*Task/}).waitFor({state:'detached'});
 await page.screenshot({path:'docs/evidence/local-calendar.png'});
 await button('Chat');await page.getByRole('textbox').waitFor();
 assert.deepEqual(errors,[]);console.log('PASS: confirmed item shows on its day, expands typed details and collapses; the back button returns to chat.');
 await browser.close();
})().catch(e=>{console.error(e);process.exit(1);});

// Run with Node and MINDER_PLAYWRIGHT pointing to an installed Playwright package.
const {chromium}=require(process.env.MINDER_PLAYWRIGHT||'/home/deck/SketchBook/Projects/AstraMoode/AstraCamillaGui/node_modules/playwright');
const assert=require('node:assert/strict');
(async()=>{
 const browser=await chromium.launch({executablePath:process.env.MINDER_CHROMIUM||'/home/deck/.cache/ms-playwright/chromium-1243/chrome-linux64/chrome',headless:true,args:['--no-sandbox']});
 const page=await browser.newPage({viewport:{width:412,height:892}});const errors=[];page.on('pageerror',e=>errors.push(e.message));
 const url=process.env.MINDER_URL||'http://127.0.0.1:8180';
 async function semantics(){await page.waitForSelector('flutter-view',{timeout:60000});await page.locator('flt-semantics-placeholder').evaluate(e=>e.click()).catch(()=>{});await page.getByRole('textbox').waitFor({timeout:30000});}
 async function button(name){await page.getByRole('button',{name,exact:true}).last().click();}
 async function board(){return page.evaluate(()=>fetch('/v1/board').then(r=>r.json()));}
 async function counts(){const b=await board();return [b.open.length,b.in_progress.length,b.done.length];}
 async function column(statusLabel){await page.getByRole('button',{name:new RegExp('^'+statusLabel+', \\d+$')}).click();await page.waitForTimeout(400);}
 await page.goto(url);await semantics();
 await button('Board');
 await page.getByRole('textbox').waitFor();
 await page.getByText('Nothing to pick up. Add a task above or tell Minder.').waitFor({timeout:15000});
 await page.getByRole('textbox').click();await page.keyboard.type('Call plumber',{delay:20});await page.keyboard.press('Enter');
 await page.getByRole('button',{name:'Start',exact:true}).waitFor({timeout:15000});
 assert.deepEqual(await counts(),[1,0,0]);
 await button('Start');
 await page.getByText('Moved to In progress').waitFor({timeout:10000});
 assert.deepEqual(await counts(),[0,1,0]);
 await button('Undo');
 await page.getByRole('button',{name:'Start',exact:true}).waitFor({timeout:10000});
 assert.deepEqual(await counts(),[1,0,0]);
 await button('Start');
 await page.getByText('Moved to In progress').waitFor({timeout:10000});
 await column('In progress');
 await page.getByRole('button',{name:'Done',exact:true}).last().waitFor({timeout:10000});
 await page.screenshot({path:'docs/evidence/local-board.png'});
 await button('Done');
 await page.getByText('Moved to Done').waitFor({timeout:10000});
 assert.deepEqual(await counts(),[0,0,1]);
 await column('Done');
 await page.getByRole('button',{name:'Reopen',exact:true}).waitFor({timeout:10000});
 await button('Reopen');
 await page.getByText('Moved to To pick up').waitFor({timeout:10000});
 assert.deepEqual(await counts(),[1,0,0]);
 const chat=await page.evaluate(()=>fetch('/v1/chat').then(r=>r.json()));
 assert.deepEqual(chat,[],'board actions must add nothing to chat');
 await button('Calendar');await page.getByRole('button',{name:/Today/}).first().waitFor({timeout:15000});
 await button('Board');await page.getByRole('textbox').waitFor({timeout:15000});
 await button('Chat');await page.getByRole('textbox').waitFor();
 assert.deepEqual(errors,[]);console.log('PASS: a task is added, started, undone, finished and reopened on the board; chat stays empty; Board, Calendar and Chat switch.');
 await browser.close();
})().catch(e=>{console.error(e);process.exit(1);});

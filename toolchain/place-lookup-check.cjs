// Run with Node, MINDER_URL and SERPER_CALLS_URL (the Serper fixture's call list).
const {chromium}=require(process.env.MINDER_PLAYWRIGHT||'/home/deck/SketchBook/Projects/AstraMoode/AstraCamillaGui/node_modules/playwright');
const assert=require('node:assert/strict');
(async()=>{
 const browser=await chromium.launch({executablePath:process.env.MINDER_CHROMIUM||'/home/deck/.cache/ms-playwright/chromium-1243/chrome-linux64/chrome',headless:true,args:['--no-sandbox']});
 const page=await browser.newPage({viewport:{width:412,height:892}});const errors=[];page.on('pageerror',e=>errors.push(e.message));
 const url=process.env.MINDER_URL;const calls=async()=>fetch(process.env.SERPER_CALLS_URL).then(r=>r.json());
 async function semantics(){await page.waitForSelector('flutter-view',{timeout:60000});await page.locator('flt-semantics-placeholder').evaluate(e=>e.click()).catch(()=>{});await page.getByRole('textbox').waitFor({timeout:30000});}
 async function send(text){await page.getByRole('textbox').click();await page.keyboard.type(text,{delay:20});await page.waitForTimeout(300);await page.getByRole('button',{name:'Send',exact:true}).click();await page.waitForTimeout(300);}
 async function has(text){const escaped=text.replace(/[.*+?^${}()|[\]\\]/g,'\\$&');await page.getByText(text,{exact:false}).or(page.getByRole('group',{name:new RegExp(escaped)})).last().waitFor({timeout:30000});}
 await page.goto(url);await semantics();
 await send('Aviad Landgraaf SnowWorld 17 to 24 november');
 await page.getByRole('button',{name:'Look it up',exact:true}).waitFor();await has('LOOK UP · ASKS FIRST');await has('Search Google Maps for "SnowWorld Landgraaf"?');
 assert.deepEqual(await calls(),[],'The offer ran a search');
 await page.screenshot({path:'docs/evidence/place-lookup-offer.png'});
 await page.getByRole('button',{name:'Look it up',exact:true}).click();
 await page.getByRole('button',{name:'Confirm',exact:true}).waitFor();await has('6372 VG Landgraaf');
 assert.deepEqual(await calls(),['/maps SnowWorld Landgraaf'],'Expected exactly the approved maps search');
 await page.screenshot({path:'docs/evidence/place-lookup-preview.png'});
 assert.equal(await page.getByRole('button',{name:'Look it up',exact:true}).count(),0,'The answered offer kept its buttons');
 await page.getByRole('button',{name:'Discard',exact:true}).click();await has('Discarded');
 await send('SnowWorld again on 25 november');
 await page.getByRole('button',{name:'Skip',exact:true}).waitFor();await page.getByRole('button',{name:'Skip',exact:true}).click();
 await page.waitForTimeout(1500);assert.deepEqual(await calls(),['/maps SnowWorld Landgraaf'],'Skip ran a search');
 assert.deepEqual(errors,[]);console.log('PASS: offer card shows Skip and Look it up; the offer ran no search; the card names the search; Look it up ran exactly that one maps search; the preview carries the postal code; Skip ran none.');
 await browser.close();
})().catch(e=>{console.error(e);process.exit(1);});

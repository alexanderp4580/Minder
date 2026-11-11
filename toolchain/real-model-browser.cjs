const {chromium}=require('/home/deck/SketchBook/Projects/AstraMoode/AstraCamillaGui/node_modules/playwright');
const assert=require('node:assert/strict');
(async()=>{const b=await chromium.launch({executablePath:'/home/deck/.cache/ms-playwright/chromium-1243/chrome-linux64/chrome',headless:true,args:['--no-sandbox']});try{
const p=await b.newPage({viewport:{width:412,height:892}});const base=process.env.MINDER_REAL_URL;
await p.goto(base);await p.waitForSelector('flutter-view',{timeout:60000});await p.locator('flt-semantics-placeholder').evaluate(el=>el.click());await p.getByRole('textbox').waitFor();
async function send(text){await p.getByRole('textbox').click();await p.keyboard.type(text,{delay:20});await p.waitForTimeout(300);await p.getByRole('button',{name:'Send',exact:true}).click();}
async function entries(){return p.request.get(base+'/v1/chat').then(r=>r.json());}
async function until(predicate){const deadline=Date.now()+90000;while(Date.now()<deadline){const es=await entries();const errors=es.filter(e=>e.kind==='error');if(errors.length)throw Error('Assistant error: '+errors[0].text);if(predicate(es))return es;await new Promise(r=>setTimeout(r,250));}throw Error('Timed out waiting for the real assistant');}
await send('Create a task titled M0 live check. Do not set a due date.');
await p.getByRole('button',{name:'Confirm',exact:true}).waitFor({timeout:90000});
let es=await entries();const preview=es.find(e=>e.kind==='preview');assert(preview);assert.equal(preview.change.action,'create_item');assert.equal(es.filter(e=>e.kind==='result').length,0);
await p.screenshot({path:'docs/evidence/real-model-preview.png'});
await p.getByRole('button',{name:'Confirm',exact:true}).click();await p.getByRole('button',{name:'Undo',exact:true}).waitFor({timeout:30000});
await send('What is open?');
es=await until(es=>es.some(e=>e.kind==='minder'&&e.text.includes('M0 live check')));console.log('Real open-item answer received.');
await p.getByRole('button',{name:'Undo',exact:true}).last().click();
await until(es=>es.some(e=>e.kind==='result'&&e.text.startsWith('Undone:')));
es=await entries();assert.equal(es.filter(e=>e.kind==='error').length,0);console.log('PASS: real OpenRouter call through Flutter web, preview, Confirm, open-item query and Undo.');
}finally{await b.close();}})().catch(e=>{console.error(e.message);process.exit(1)});

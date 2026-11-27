import test from 'node:test';
import assert from 'node:assert/strict';
import {createServer} from 'node:http';
import {readFile} from 'node:fs/promises';
import {resolve,extname,sep} from 'node:path';
import {chromium} from '../out/ui-test/node_modules/playwright/index.mjs';
const root=resolve('desktop/frontend/dist');
const types={'.html':'text/html','.js':'text/javascript','.css':'text/css','.svg':'image/svg+xml'};
async function setup(t){
 const server=createServer(async(req,res)=>{
  try{
   const url=new URL(req.url,'http://localhost').pathname;
   if(url==='/wails/runtime.js'){res.writeHead(200,{'Content-Type':'text/javascript'});res.end("export const Call={ByName:(name,...args)=>window.bridge(name.split('.').at(-1),...args)}");return}
   const file=resolve(root,'.'+(url==='/'?'/index.html':url));
   if(!file.startsWith(root+sep)){res.writeHead(403);res.end();return}
   const data=await readFile(file);res.writeHead(200,{'Content-Type':types[extname(file)]||'text/plain'});res.end(data);
  }catch{res.writeHead(404);res.end()}
 });
 await new Promise(r=>server.listen(0,'127.0.0.1',r));
 const browser=await chromium.launch({headless:true});
 t.after(async()=>{await browser.close();await new Promise(r=>server.close(r))});
 const page=await browser.newPage({viewport:{width:1100,height:800}});
 const errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.addInitScript(()=>{
  window.prompt=()=>{throw Error('native prompt forbidden')};
  window.confirm=()=>{throw Error('native confirm forbidden')};
  const root='/work';
  const source=name=>({name,path:root+'/'+name,directory:root});
  const state={name:'A.c',names:['A.c'],calls:[],cases:[],revision:'missing',observing:false,failObserve:false,failCreate:false,run:null,holdRun:false,finishRun:null};
  window.fixture=state;
  const nav=()=>({root,source:source(state.name)});
  window.bridge=async(method,...args)=>{
   state.calls.push([method,...args]);
   switch(method){
   case 'Version':return '0.1.0-beta.2';
   case 'Config':return {c:'',cpp:'',editor:'',root,selected:'A.c'};
   case 'ConfigWarning':return '';
   case 'Initial':return nav();
   case 'List':return state.names.map(name=>({...source(name),directory:false}));
   case 'LoadCases':return {cases:state.cases,revision:state.revision};
   case 'Cached':return null;
   case 'IsCurrent':return true;
   case 'Progress':return {phase:'running',completed:1,total:2};
   case 'RunBatch':if(state.holdRun)return new Promise(r=>{state.finishRun=()=>r(state.run)});return state.run;
   case 'Cancel':state.finishRun?.();return;
   case 'Select':state.name=args[0].split('/').at(-1);return nav();
   case 'Orphans':return [];
   case 'CheckSelected':return true;
   case 'TakeDrop':return null;
   case 'Invalidate':case 'StoreConfig':return;
   case 'Create':
    if(state.failCreate)throw Error('文件已存在');
    state.name=args[1];state.names.push(state.name);state.cases=[];state.revision='missing';return nav();
   case 'Rename':
    state.names=state.names.map(n=>n===state.name?args[0]:n);state.name=args[0];return nav();
   case 'SaveCases':state.cases=structuredClone(args[0]);state.revision='saved';return {cases:state.cases,revision:state.revision};
   case 'Observe':if(state.failObserve)throw Error('Observer 尚未安装');state.observing=true;return;
   case 'ObservationState':return state.observing?'running':'closed';
   default:throw Error('unexpected RPC '+method);
   }
  };
 });
 await page.goto('http://127.0.0.1:'+server.address().port);
 await page.waitForFunction(()=>!document.querySelector('#new').disabled&&!document.querySelector('#rename').disabled);
 t.after(()=>assert.deepEqual(errors,[]));
 return page;
}
test('new file and rename use embedded dialogs, cancel and Unicode names',async t=>{
 const p=await setup(t);
 await p.locator('#new').click();await p.locator('#actionDialog').waitFor();
 await p.locator('#actionCancel').click();await p.locator('#actionDialog').waitFor({state:'hidden'});
 await p.waitForFunction(()=>!document.querySelector('#new').disabled);
 assert.equal(await p.evaluate(()=>fixture.calls.filter(c=>c[0]==='Create').length),0);
 await p.locator('#new').click();await p.locator('#actionName').fill('测试 B.c');await p.locator('#actionName').press('Enter');
 await p.waitForFunction(()=>document.querySelector('#title').textContent==='测试 B.c'&&!document.querySelector('#rename').disabled);
 assert.deepEqual(await p.evaluate(()=>fixture.calls.find(c=>c[0]==='Create')),['Create','/work','测试 B.c']);
 await p.getByRole('button',{name:'当前文件操作'}).click();await p.locator('#rename').click();await p.locator('#actionName').fill('改名.c');await p.locator('#actionAccept').click();
 await p.waitForFunction(()=>document.querySelector('#title').textContent==='改名.c'&&!document.querySelector('#new').disabled);
 assert.deepEqual(await p.evaluate(()=>fixture.calls.find(c=>c[0]==='Rename')),['Rename','改名.c']);
 await p.evaluate(()=>fixture.failCreate=true);
 await p.locator('#new').click();await p.locator('#actionAccept').click();
 await p.waitForFunction(()=>document.querySelector('#status').textContent.includes('文件已存在')&&!document.querySelector('#new').disabled);
 assert.equal(await p.locator('#title').textContent(),'改名.c');
});
test('observe saves current sample, starts without confirmation and surfaces error',async t=>{
 const p=await setup(t);
 await p.locator('#add').click();await p.locator('#input').fill('7 8');
 await p.locator('#observe').click();
 await p.waitForFunction(()=>document.querySelector('#observerState').textContent.includes('正在运行'));
 const calls=await p.evaluate(()=>fixture.calls.filter(c=>['SaveCases','Observe'].includes(c[0])));
 assert.deepEqual(calls,[['SaveCases',[{input:'7 8',expected:null}],'missing'],['Observe','','saved',0]]);
 await p.evaluate(()=>fixture.observing=false);
 await p.waitForFunction(()=>document.querySelector('#observerState').textContent.includes('已关闭'));
 await p.evaluate(()=>fixture.failObserve=true);
 await p.locator('#observe').click();
 await p.waitForFunction(()=>document.querySelector('#observerState').textContent.includes('尚未安装')&&!document.querySelector('#observe').disabled);
});
test('cancel dirty navigation and delete confirmation preserve samples',async t=>{
 const p=await setup(t);
 await p.locator('#add').click();await p.locator('#input').fill('unsaved');
 await p.getByRole('button',{name:'当前文件操作'}).click();await p.locator('#rename').click();await p.locator('#actionDialog').waitFor();
 await p.keyboard.press('Escape');await p.waitForFunction(()=>!document.querySelector('#rename').disabled);
 assert.equal(await p.locator('#input').inputValue(),'unsaved');
 assert.equal(await p.evaluate(()=>fixture.calls.filter(c=>c[0]==='Rename').length),0);
 await p.locator('#caseMenu summary').click();await p.locator('#remove').click();await p.locator('#actionCancel').click();
 await p.waitForFunction(()=>!document.querySelector('#remove').disabled);
 assert.equal(await p.locator('#input').inputValue(),'unsaved');
 await p.locator('#caseMenu summary').click();await p.locator('#remove').click();await p.locator('#actionAccept').click();
 await p.locator('#empty').waitFor();
 assert.equal(await p.evaluate(()=>fixture.calls.filter(c=>c[0]==='SaveCases').length),0);
});

test('long file tree has independent scrolling and a resizable sidebar',async t=>{
 const p=await setup(t);
 await p.evaluate(()=>fixture.names.push(...Array.from({length:130},(_,i)=>'long-file-'+i+'.c')));
 await p.locator('#folderMenu summary').click();await p.locator('#refreshTree').click();
 await p.waitForFunction(()=>document.querySelectorAll('.tree-item').length===131);
 const initial=await p.locator('#title').boundingBox();
 await p.locator('#file').evaluate(e=>e.scrollTop=600);
 const after=await p.locator('#title').boundingBox();
 assert.equal(initial.y,after.y);assert.ok(initial.y<100);
 const scroll=await p.evaluate(()=>({tree:document.querySelector('#file').scrollTop,body:document.scrollingElement.scrollTop,main:document.querySelector('#workspace').scrollTop}));
 assert.ok(scroll.tree>0);assert.equal(scroll.body,0);assert.equal(scroll.main,0);
 const grip=p.locator('#sidebarGrip');await grip.focus();await p.keyboard.press('ArrowRight');assert.equal(await grip.getAttribute('aria-valuenow'),'232');
});
test('batch results follow selected sample, compare once, and edits invalidate prior status without auto-save',async t=>{
 const p=await setup(t);
 await p.locator('#add').click();await p.locator('#input').fill('input-one');await p.locator('#compare').check();await p.locator('#expected').fill('8');
 await p.locator('#add').click();await p.locator('#input').fill('input-two');await p.locator('#compare').check();await p.locator('#expected').fill('1 2\n4 5');
 await p.evaluate(()=>{
  const process=(stdout)=>({stdout,stderr:'',exitCode:0,elapsedMs:12,state:'ok'});
  fixture.run={runId:'r1',source:{name:'A.c',path:'/work/A.c',directory:'/work'},sourceDigest:'d',compiler:'gcc',state:'completed',compilation:process(''),diagnostics:[],stale:false,cases:[
   {index:0,input:'input-one',expected:'8',process:process('8\n'),state:'passed'},
   {index:1,input:'input-two',expected:'1 2\n4 5',process:process('1 2\n4 6\n'),state:'mismatch',difference:{line:2,column:3,expected:'5',actual:'6'}}
  ]};fixture.holdRun=true;
 });
 assert.equal(await p.evaluate(()=>fixture.calls.filter(c=>c[0]==='SaveCases').length),0);
 await p.locator('#all').click();await p.locator('#cancel').waitFor();assert.equal(await p.locator('#run').isVisible(),false);
 await p.waitForFunction(()=>!!fixture.finishRun);await p.evaluate(()=>fixture.finishRun());
 await p.waitForFunction(()=>document.querySelector('#resultTitle').textContent==='输出不一致'&&!document.querySelector('#all').disabled);
 assert.equal(await p.locator('#results .output-table tbody tr').count(),2);
 assert.equal(await p.locator('#results .bad-row').count(),1);
 assert.equal(await p.locator('#results .diffmark').count(),2);
 assert.equal(await p.locator('#diagnosticsSection').isVisible(),false);
 assert.match(await p.locator('#tabs').textContent(),/✓.*≠/);
 await p.locator('#rawToggle').click();assert.equal(await p.locator('#results .output-table').count(),0);
 assert.match(await p.locator('#results').textContent(),/4 6/);
 await p.getByRole('button',{name:'样例 1 · 通过'}).click();
 assert.equal(await p.locator('#input').inputValue(),'input-one');
 assert.equal(await p.locator('#resultTitle').textContent(),'通过');
 assert.equal(await p.locator('#results pre').count(),1);
 assert.doesNotMatch(await p.locator('#results').textContent(),/input-two|4 6/);
 await p.locator('#input').fill('changed');
 assert.equal(await p.locator('#stale').isVisible(),true);
 assert.equal(await p.locator('#results .snapshot').count(),1);
 assert.equal(await p.evaluate(()=>fixture.calls.filter(c=>c[0]==='SaveCases').length),1);
 assert.equal(await p.locator('#saved').textContent(),'未保存');
});

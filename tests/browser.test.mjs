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
 t.after(()=>new Promise(r=>server.close(r)));
 const browser=await chromium.launch({headless:true});
 t.after(()=>browser.close());
 const page=await browser.newPage({viewport:{width:1100,height:800}});
 const errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.addInitScript(()=>{
  window.prompt=()=>{throw Error('native prompt forbidden')};
  window.confirm=()=>{throw Error('native confirm forbidden')};
  const root='/work';
  const source=name=>({name,path:root+'/'+name,directory:root});
  const state={name:'A.c',names:['A.c'],calls:[],cases:[],revision:'missing',observing:false,failObserve:false,failCreate:false};
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
 await p.locator('#rename').click();await p.locator('#actionName').fill('改名.c');await p.locator('#actionAccept').click();
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
 await p.locator('#rename').click();await p.locator('#actionDialog').waitFor();
 await p.keyboard.press('Escape');await p.waitForFunction(()=>!document.querySelector('#rename').disabled);
 assert.equal(await p.locator('#input').inputValue(),'unsaved');
 assert.equal(await p.evaluate(()=>fixture.calls.filter(c=>c[0]==='Rename').length),0);
 await p.locator('#remove').click();await p.locator('#actionCancel').click();
 await p.waitForFunction(()=>!document.querySelector('#remove').disabled);
 assert.equal(await p.locator('#input').inputValue(),'unsaved');
 await p.locator('#remove').click();await p.locator('#actionAccept').click();
 await p.locator('#empty').waitFor();
 assert.equal(await p.evaluate(()=>fixture.calls.filter(c=>c[0]==='SaveCases').length),0);
});

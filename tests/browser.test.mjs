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
   case 'OJCheckAdapter':case 'OJAccountStatus':return {available:state.ojAvailable??false,state:state.ojAvailable?'signed_in':'unavailable'};
   case 'OJBinding':return state.name==='A.c'?(state.ojBinding??null):null;
   case 'OJCachedJudges':return state.ojJudges??{};
   case 'OJWorkspace':return {items:state.ojBinding?[{path:'/work/A.c',directory:'/work',group:'work',binding:state.ojBinding}]:[],limited:false};
   case 'ReadLocalDocument':return state.document??null;
   case 'OJRecentContests':return [];
   case 'OJPollReceipt':return {id:args[0],result:'AC',language:'C'};
   case 'OJRefreshJudges':return 1;
   case 'OJUnbind':state.ojBinding=null;state.ojJudges={};return;
   case 'OJSubmissionAvailable':return !!state.ojAvailable&&!!state.ojBinding;
   case 'KitStatus':return {version:'0.6.10',path:''};
   case 'ObserverStatus':return {available:false};
   case 'Prepare':return {source:source(state.name),text:'int main(void){return 0;}',digest:'fixture-digest',verified:false};
   case 'OJPrepareSubmission':return {plan:{token:'fixture-token',target:{kind:'problemset',problemId:'42'},accountId:'7',languages:['C']},title:'测试题',label:'A',nickname:'fixture'};
   case 'OJSendSubmission':return {id:'12',result:'AC',accountId:'7',language:'C'};
   case 'Version':return '0.1.0-beta.5';
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
   case 'ChooseCreateDirectory':return '/other';
   case 'DetectEditors':return [{name:'VS Code',path:'/apps/code'}];
   case 'OpenManual':return {origin:'installed',fingerprint:'fixture-v1',reference:{version:'0.6.10',notes:'## 使用约定\n- 示例',sections:[{title:'输入与输出',entries:['read_number']}],categories:[{title:'数字读写',entries:[{id:'read_number',title:'读取数字',description:'读取一个整数。',example:'int a = read_int();',signatures:['int read_int(void);']}]}]}};
   case 'SearchManual':return [{id:'read_number',category:'数字读写'}];
   case 'CopyManualSignature':case 'CopyManualExample':return;
   case 'OpenEditor':return;
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

test('explicit new location and editor detection do not depend on system default',async t=>{
 const p=await setup(t);
 await p.locator('#new').click();assert.equal(await p.locator('#actionDirectory').textContent(),'/work');
 await p.locator('#actionBrowse').click();await p.waitForFunction(()=>document.querySelector('#actionDirectory').textContent==='/other');
 await p.locator('#actionCancel').click();await p.waitForFunction(()=>!document.querySelector('#new').disabled);
 assert.equal(await p.evaluate(()=>fixture.calls.filter(c=>c[0]==='Create').length),0);
 await p.locator('#openEditor').click();await p.waitForFunction(()=>document.querySelector('#openEditor').textContent.includes('VS Code'));
 assert.deepEqual(await p.evaluate(()=>fixture.calls.find(c=>c[0]==='OpenEditor')),['OpenEditor','/apps/code']);
});

test('manual preserves version-scoped reading state and copies a signature',async t=>{
 const p=await setup(t);
 await p.locator('#manualBtn').click();await p.waitForFunction(()=>!document.querySelector('#manualSearch').disabled);
 await p.locator('#manualNav summary').click();await p.locator('#manualNav [data-entry]').click();
 assert.equal(await p.locator('#manualDetail h2').textContent(),'读取数字');
 assert.ok(await p.locator('#manualDetail .syntax-keyword').count()>0);
 await p.getByRole('button',{name:'复制函数接口'}).click();
 await p.waitForFunction(()=>fixture.calls.some(c=>c[0]==='CopyManualSignature'));
 assert.deepEqual(await p.evaluate(()=>fixture.calls.find(c=>c[0]==='CopyManualSignature')),['CopyManualSignature','read_number','','fixture-v1','int read_int(void);']);
 await p.locator('#manualClose').click();await p.locator('#manualBtn').click();
 await p.waitForFunction(()=>document.querySelector('#manualDetail h2')?.textContent==='读取数字'&&!document.querySelector('#manualSearch').disabled);
 assert.equal(await p.locator('#manualNav details').getAttribute('open'),'');
 await p.locator('#manualSearch').fill('读取');await p.waitForFunction(()=>document.querySelector('#manualNav [data-entry] small'));
 await p.locator('#manualSearch').press('ArrowDown');await p.locator('#manualSearch').press('Enter');
 assert.equal(await p.locator('#manualDetail h2').textContent(),'读取数字');
});

test('OJ cached state survives keyed tree refresh, explicit refresh is the only online read',async t=>{
 const p=await setup(t);
 await p.evaluate(()=>{fixture.ojAvailable=true;fixture.ojBinding={title:'测试题',label:'A',sourceUrl:'https://accoding.buaa.edu.cn/',target:{kind:'problemset'}};fixture.ojJudges={'/work/A.c':{status:{accepted:true,latest:{id:'12',result:'WA',language:'C'}},checked:'2026-11-27T00:00:00Z'}};fixture.names.push('B.c')});
 await p.locator('#settings').click();await p.waitForFunction(()=>!document.querySelector('#ojLogin').disabled);await p.locator('#closeConfig').click();
 await p.locator('#folderMenu summary').click();await p.locator('#refreshTree').click();
 assert.equal(await p.locator('.tree-judge:not([hidden])').count(),0);
 await p.evaluate(()=>window.originalTreeButton=document.querySelector('.tree-item'));
 await p.evaluate(()=>fixture.names.push('C.c'));
 await p.waitForFunction(()=>document.querySelectorAll('.tree-item').length===3);
 assert.equal(await p.evaluate(()=>window.originalTreeButton===document.querySelector('.tree-item')),true);
 assert.equal(await p.evaluate(()=>fixture.calls.filter(c=>c[0]==='OJRefreshJudges').length),0);
 await p.locator('.tree-item').filter({hasText:'B.c'}).click();await p.waitForFunction(()=>document.querySelector('#title').textContent==='B.c'&&!document.querySelector('#new').disabled);
 await p.locator('.tree-item').filter({hasText:'A.c'}).click();await p.waitForFunction(()=>!document.querySelector('#new').disabled);await p.locator('#ojMode').click();await p.waitForFunction(()=>!document.querySelector('#ojBinding').hidden);
 assert.match(await p.locator('#ojJudgeText').textContent(),/曾 AC.*WA/);
 await p.locator('#ojRefreshCurrent').click();await p.waitForFunction(()=>fixture.calls.some(c=>c[0]==='OJRefreshJudges'));
 assert.deepEqual(await p.evaluate(()=>fixture.calls.filter(c=>c[0]==='OJRefreshJudges')), [['OJRefreshJudges',['/work/A.c']]]);
 await p.waitForFunction(()=>!document.querySelector('#ojUnbind').disabled);
 await p.locator('#ojUnbind').click();await p.locator('#actionAccept').click();await p.waitForFunction(()=>document.querySelector('#ojBinding').hidden&&!document.querySelector('.tree-judge:not([hidden])'));
});

test('online submission in unified preview requires explicit language and confirmation',async t=>{
 const p=await setup(t);
 await p.evaluate(()=>{fixture.ojAvailable=true;fixture.ojBinding={title:'测试题',label:'A',target:{kind:'problemset'}}});
 await p.locator('#settings').click();await p.waitForFunction(()=>!document.querySelector('#ojLogin').disabled);await p.locator('#closeConfig').click();await p.locator('#ojMode').click();await p.locator('#ojSubmitBtn').click();await p.locator('#ojSubmitPrepare').waitFor();
 assert.equal(await p.evaluate(()=>fixture.calls.filter(c=>c[0]==='OJSendSubmission').length),0);
 await p.locator('#ojSubmitPrepare').click();await p.locator('#ojSubmitConfirm').waitFor();
 assert.equal(await p.locator('#ojSubmitConfirm').isDisabled(),true);
 await p.locator('#ojSubmitLanguage').selectOption('C');await p.locator('#ojSubmitConfirm').click();
 await p.waitForFunction(()=>document.querySelector('#ojSubmitNotice').textContent.includes('#12'));
 assert.deepEqual(await p.evaluate(()=>fixture.calls.filter(c=>c[0]==='OJSendSubmission')), [['OJSendSubmission','','fixture-digest','fixture-token','C']]);
});

test('unavailable Adapter hides all OJ entry points even for an existing binding',async t=>{
 const p=await setup(t);
 await p.evaluate(()=>{fixture.ojAvailable=true;fixture.ojBinding={title:'题',label:'A',target:{kind:'problemset'}};fixture.ojJudges={'/work/A.c':{status:{accepted:true},checked:'2026-11-27T00:00:00Z'}}});
 await p.locator('#settings').click();await p.waitForFunction(()=>document.querySelector('#ojAccountState').textContent.includes('已登录'));
 await p.locator('#closeConfig').click();await p.locator('#ojMode').click();await p.waitForFunction(()=>!document.querySelector('#ojBinding').hidden&&!!document.querySelector('.oj-problem'));
 await p.evaluate(()=>fixture.ojAvailable=false);
 await p.locator('#settings').click();await p.waitForFunction(()=>document.querySelector('#ojLogin').disabled&&!document.querySelector('#ojCheckAgain').disabled);
 await p.locator('#closeConfig').click();await p.waitForFunction(()=>['ojImport','ojRefreshTree','ojBinding','ojBindingError','ojJudgeInfo'].every(id=>document.getElementById(id).hidden)&&!document.querySelector('.tree-judge:not([hidden])'));
 await p.locator('#copy').click();await p.locator('#preview').waitFor();assert.equal(await p.locator('#ojSubmitArea').isVisible(),false);assert.equal(await p.locator('#copyConfirm').isEnabled(),true);await p.locator('#closePreview').click();
 await p.evaluate(()=>fixture.ojAvailable=true);
 await p.locator('#settings').click();await p.waitForFunction(()=>!document.querySelector('#ojCheckAgain').disabled);await p.locator('#ojCheckAgain').click();await p.waitForFunction(()=>!document.querySelector('#ojLogin').disabled);
 await p.locator('#closeConfig').click();await p.locator('#ojMode').click();await p.waitForFunction(()=>!document.querySelector('#ojBinding').hidden);
 assert.equal(await p.evaluate(()=>fixture.calls.filter(c=>c[0]==='OJUnbind'||c[0]==='OJRefreshJudges').length),0);
});

test('offline document renders Markdown and math, no external image or HTML execution',async t=>{
 const p=await setup(t);await p.evaluate(()=>fixture.document={name:'problem.md',content:'# 本地题面\n\n$x^2$\n\n![test](https://invalid.example/image.png)\n\n<script>throw Error("unsafe")</script>\n\n    1 2\n'});
 await p.locator('#new').click();await p.locator('#actionCancel').click(); // refresh via source selection
 await p.evaluate(()=>fixture.names.push('B.c'));await p.locator('.tree-item').filter({hasText:'B.c'}).click();await p.locator('#documentBtn').click();
 await p.locator('#documentDialog .katex').waitFor();assert.equal(await p.locator('#documentBody img,#documentBody script').count(),0);
 assert.match(await p.locator('#documentBody').textContent(),/此处有一张图片/);assert.equal(await p.locator('#ojMode').isVisible(),false)
});
test('detailed judge table uses actual line structure and concise signal guidance',async t=>{
 const p=await setup(t);await p.evaluate(()=>{fixture.ojAvailable=true;fixture.ojBinding={title:'测试题',label:'A',target:{kind:'contest',contestId:'42',problemId:'5'}};fixture.ojJudges={'/work/A.c':{status:{accepted:false,latest:{id:'12',result:'REG',language:'C',score:'0.5',detail:'12\nAccepted | 1 * (1 / 2) | 1 ms | 1852 KB\nRuntime Error (SIGSEGV) | 0 * (1 / 2) | 140 ms | 1752 KB\n'}},checked:'2026-11-27T00:00:00Z'}}});
 await p.locator('#settings').click();await p.waitForFunction(()=>!document.querySelector('#ojLogin').disabled);await p.locator('#closeConfig').click();await p.locator('#ojMode').click();await p.locator('#ojDetailBtn').click();
 assert.match(await p.locator('#ojDetailBody').textContent(),/SIGSEGV.*非法内存访问/);assert.equal(await p.locator('#ojDetailBody tr').count(),3)
});

test('ordinary copy remains offline even when OJ is installed',async t=>{
 const p=await setup(t);await p.evaluate(()=>{fixture.ojAvailable=true;fixture.ojBinding={title:'题',label:'A',target:{kind:'problemset',problemId:'5'}}});
 await p.locator('#settings').click();await p.waitForFunction(()=>!document.querySelector('#ojLogin').disabled);await p.locator('#closeConfig').click();
 await p.locator('#copy').click();assert.equal(await p.locator('#ojSubmitArea').isVisible(),false);
 assert.equal(await p.evaluate(()=>fixture.calls.filter(c=>c[0]==='OJSubmissionAvailable').length),0)
});

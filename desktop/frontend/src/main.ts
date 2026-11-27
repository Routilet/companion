import {installOJStatus} from "./oj-status.js";
import {installOJSubmit} from "./oj-submit.js";
import {installOJImport} from "./oj-import.js";
import {installKitSwitch} from "./kit-switch.js";
import {installOJAccount} from "./oj-account.js";
import {installWorkbench} from "./oj-workbench.js";
import {installDocument} from "./local-document.js";
import {installManual} from "./manual.js";
import {installDialogs} from "./dialogs.js";
export {};
type Source={path:string;directory:string;name:string};
type Nav={root:string;source:Source};
type Entry={path:string;name:string;directory:boolean};
type Orphan={name:string;count:number};
type Case={input:string;expected:string|null};
type CaseSet={cases:Case[];revision:string};
type Process={stdout:string;stderr:string;exitCode:number;elapsedMs:number;state:string};
type Part={text:string;changed:boolean};
type Difference={line:number;column:number;expected:string;actual:string;expectedParts?:Part[];actualParts?:Part[];coarse?:boolean};
type ObserverInfo={version:string;backend:string;executable:string;error?:string};
type KitStatus={version:string;path:string;error?:string};
type Settings={root:string;selected:string;c:string;cpp:string;editor:string};
type Progress={phase:string;completed:number;total:number;runId:string};
type Submission={source:Source;text:string;digest:string;verified:boolean;kitVersion?:string;components?:string[]};
type CaseResult={index:number;input:string;expected:string|null;process:Process;state:string;difference?:Difference};
type Diagnostic={severity:string;message:string;raw:string;file?:string;line?:number;column?:number};
type Result={runId:string;source:Source;sourceDigest:string;kitVersion?:string;components?:string[];compiler:string;state:string;compilation:Process;cases:CaseResult[];diagnostics:Diagnostic[];stale:boolean};
type Runtime={Call:{ByName(name:string,...args:unknown[]):Promise<unknown>}};
let runtime:Runtime|undefined,source:Source|undefined,busy=false,dirty=false,loaded=false,selected=0,revision="missing",cases:Case[]=[],last:Result|undefined;
const el=<T extends HTMLElement=HTMLElement>(id:string)=>document.getElementById(id) as T;
const btn=(id:string)=>el<HTMLButtonElement>(id);
const dialogs=installDialogs();
const input=el<HTMLTextAreaElement>("input"),expected=el<HTMLTextAreaElement>("expected"),compare=el<HTMLInputElement>("compare"),compiler=el<HTMLInputElement>("compiler");
const expansionByRoot=new Map<string,Set<string>>();
function restoreExpansion(path:string):string[]{try{const v=JSON.parse(localStorage.getItem("companion.expanded")??"null");return v?.root===path&&Array.isArray(v.paths)?v.paths.filter((p:unknown)=>typeof p==="string").slice(0,128):[]}catch{return []}}
function rememberExpansion(){try{if(root)localStorage.setItem("companion.expanded",JSON.stringify({root,paths:[...expanded].slice(0,128)}))}catch{/* optional UI state */}}
let root="",expanded=new Set<string>(),treeEntries=new Map<string,Entry[]>(),refreshing=false,navEpoch=0,seenMissing=new Set<string>();
const compilerByLanguage:Record<string,string>={c:"",cpp:""};
let observing=false,observerPolling=false,rawResult=false;
let editorPath="",prepared:Submission|undefined,runningTest=false,dropCheck=false,progressCheck=false;
const language=(s:Source)=>/\.c$/i.test(s.name)?"c":"cpp";
const status=el("status"),result=el("result"),stale=el("stale");
const noticeObserver=new MutationObserver(()=>{el("observerState").hidden=!el("observerState").textContent;status.hidden=!status.textContent});noticeObserver.observe(el("observerState"),{childList:true,characterData:true,subtree:true});noticeObserver.observe(status,{childList:true,characterData:true,subtree:true});
const labels:Record<string,string>={completed:"测试完成",passed:"通过",mismatch:"输出不一致",ran:"已运行（无预期）",compilation_failed:"编译失败",runtime_failed:"运行失败",timeout:"超时",cancelled:"已取消",output_limit:"输出超限"};
async function call<T>(method:string,...args:unknown[]):Promise<T>{
 if(!runtime)throw Error("未连接 Companion 本地后端");
 return await runtime.Call.ByName("main.Desktop."+method,...args) as T;
}
function markStale(){if(last&&!last.stale){last.stale=true;renderTabs();renderResult()}}
let saveTimer:ReturnType<typeof setTimeout>|undefined,saveFlight:Promise<void>|undefined,editGeneration=0,saveError="",editingExpected=false;
function queueSave(){clearTimeout(saveTimer);saveTimer=setTimeout(()=>{if(dirty&&loaded){if(busy||saveFlight)queueSave();else void save().catch(()=>{})}},1000)}
function changes(){dirty=true;editGeneration++;markStale();updateControls();queueSave()}
function updateControls(){
 btn("open").disabled=btn("openFolder").disabled=btn("settings").disabled=busy||!runtime;btn("manualBtn").disabled=busy||!runtime;for(const id of ["openEditor","reveal","copy","previewSource"])btn(id).disabled=busy||!source;btn("settingsEditor").disabled=busy;btn("kitCurrent").disabled=busy||!source||language(source)!=="c";btn("kitRoot").disabled=busy||!root;btn("kitContest").disabled=busy||!workbench.groupPaths().length;btn("new").disabled=busy||!runtime;btn("refreshTree").disabled=busy||!root;btn("locateFile").disabled=busy||!source;btn("collapseTree").disabled=btn("newRootFolder").disabled=busy||!root;btn("ojImport").disabled=busy||!root;btn("ojRefreshTree").disabled=busy||!root;btn("ojRefreshCurrent").disabled=busy||!source;btn("ojSubmitBtn").disabled=busy||!source;btn("ojMode").disabled=btn("filesMode").disabled=busy;btn("ojOpenOriginal").disabled=btn("ojUnbind").disabled=busy||!source;btn("rename").disabled=busy||!source||!loaded;btn("import").disabled=busy||!source||!loaded;
 for(const id of ["add","save","reload","remove","run","all"])btn(id).disabled=busy||!source||(!loaded&&id!=="reload");
 btn("observe").disabled=busy||!source||!loaded||!(/\.c$/i.test(source.name))||observing;btn("save").disabled||=!dirty;btn("save").hidden=!saveError;btn("remove").disabled||=!cases.length;btn("add").disabled||=cases.length>=32;
 btn("cancel").disabled=!runningTest;btn("cancel").hidden=!runningTest;btn("run").hidden=runningTest||cases.length<2;btn("all").hidden=runningTest;input.disabled=compare.disabled=compiler.disabled=busy||!loaded;expected.disabled=busy||!loaded||!compare.checked;
 workbench.controls();el("saved").textContent=!loaded?"读取失败":saveError?"保存失败："+saveError:saveFlight?"保存中":dirty?"未保存":"";el("saved").title=dirty?"样例自动保存；失败时保留编辑内容。":"";for(const b of Array.from(el("tabs").querySelectorAll<HTMLButtonElement>("button")))b.disabled=busy;
}
function caseResult(index:number){return last?.cases.find(c=>c.index===index)}
function matches(c:CaseResult,index:number){const current=cases[index]??{input:"",expected:null};return current.input===c.input&&current.expected===c.expected}
function renderTabs(){
 const tabs=el("tabs"),scroll=tabs.scrollLeft;tabs.replaceChildren();
 const list=cases.length?cases:[{input:"",expected:null}];
 list.forEach((_,i)=>{const c=caseResult(i),b=document.createElement("button");b.className="case-tab";b.textContent=cases.length?"样例 "+(i+1):"空输入";b.setAttribute("aria-selected",String(i===selected));b.disabled=busy;
 const state=c?(last?.stale||!matches(c,i)?"stale":c.state):"unrun";
 const icons:Record<string,string>={passed:"✓",mismatch:"≠",ran:"•",stale:"◷",unrun:"—"};
 const label=state==="stale"?"结果已过期":state==="unrun"?"尚未运行":labels[state]??state;
 b.append(node("span",icons[state]??"!","case-state "+state));b.title=label;b.setAttribute("aria-label",(cases.length?"样例 "+(i+1):"空输入")+" · "+label);
 b.onclick=()=>{selected=i;rawResult=false;editingExpected=false;renderCases();el("tabs").querySelector<HTMLButtonElement>('[aria-selected="true"]')?.focus()};tabs.append(b)});
 tabs.scrollLeft=scroll;
}
function renderCases(){
 renderTabs();const c=cases[selected];el("empty").hidden=!!c;el("editor").hidden=!c;
 input.value=c?.input??"";compare.checked=c?.expected!==null&&!!c;expected.value=c?.expected??"";
 btn("run").textContent=c?"运行本组":"空输入运行";btn("run").hidden=runningTest||cases.length<2;btn("all").textContent=cases.length?"运行全部":"运行";updateControls();renderResult();
}
function applySet(s:CaseSet){cases=s.cases??[];revision=s.revision;dirty=false;saveError="";loaded=true;selected=Math.min(selected,Math.max(0,cases.length-1));renderCases();}
async function reload(){loaded=false;updateControls();try{applySet(await call<CaseSet>("LoadCases"));status.textContent=""}catch(e){status.textContent=String(e)}finally{updateControls()}}
async function select(nav:Nav){
 if(!nav.root)return;
 navEpoch++;clearTimeout(saveTimer);saveError="";editingExpected=false;
 if(root!==nav.root){workbench.invalidate();if(root)expansionByRoot.set(root,new Set(expanded));root=nav.root;expanded=new Set(expansionByRoot.get(root)??[root,...restoreExpansion(root)]);treeEntries.clear();seenMissing.clear()}
 el("folder").textContent=root.split(/[\\/]/).filter(Boolean).at(-1)??root;el("folder").title=root;
 el("workspace").scrollTop=0;closeFileMenu();rawResult=false;status.textContent="";if(!observing)el("observerState").textContent="";
 source=nav.source.path?nav.source:undefined;cases=[];selected=0;last=undefined;dirty=false;loaded=false;result.hidden=true;el("orphans").replaceChildren();
 el("title").textContent=source?.name??"选择一个源文件";el("path").textContent=source?source.directory.slice(root.length).replace(/^[\\/]/,"")||el("folder").textContent:root;el("path").title=source?.path??root;el("work").hidden=!source;el("fileActions").hidden=!source;el("batchSummary").textContent="";
 if(source){let dir=source.directory;while(dir&&dir!==root){expanded.add(dir);const next=dir.replace(/[\\/][^\\/]+$/,"");if(next===dir||next.length<root.length)break;dir=next}compiler.value=compilerByLanguage[language(source)];const wasMissing=seenMissing.has(source.path);if(wasMissing)await call("Invalidate",source.path);await reload();
  if(seenMissing.has(source.path)&&revision!=="missing"){if(!await dialogs.confirm("此路径曾消失，存在旧测试数据。是否继续使用这些样例？")){loaded=false;status.textContent="未确认旧样例；请重新选择文件";}else seenMissing.delete(source.path)}
  if(loaded){const cached=await call<Result|null>("Cached",compiler.value.trim());if(cached){if(seenMissing.has(source.path))cached.stale=true;show(cached)}await showOrphans()}
 }
 await documentReader.refresh(source?.path??"");await refreshTree(true);if(workbench.active()&&!ojAccount.connecting())await syncOJWorkspace(false);else if(workbench.active())workbench.draw();updateControls();
}
async function showOrphans(){
 const box=el("orphans");box.replaceChildren();if(!source||cases.length||dirty||!loaded)return;
 try{const list=await call<Orphan[]|null>("Orphans");if(!list?.length)return;box.append(node("p","发现未关联的测试数据","muted"));
 for(const o of list){const b=document.createElement("button");b.textContent=o.name+" · "+o.count+" 组 · 恢复";b.onclick=()=>void navigate(async()=>{if(!await dialogs.confirm("复制旧样例到当前文件？原数据保留。"))return;applySet(await call<CaseSet>("Restore",o.name,revision));markStale();await showOrphans()});box.append(b)}
 }catch(e){box.append(node("p",String(e),"muted"))}
}
const fileNodes=new Map<string,{wrap:HTMLElement;row:HTMLElement;button:HTMLButtonElement;arrow:HTMLElement;icon:HTMLElement;name:HTMLElement;more:HTMLButtonElement;child:HTMLElement;judge:HTMLElement}>();
const natural=new Intl.Collator(undefined,{numeric:true,sensitivity:"base"});
function drawTree(){
 const host=el("file"),scroll=host.scrollTop,seen=new Set<string>();
 function branch(dir:string,container:HTMLElement,depth:number){
  if(depth>24)return;
  const items=[...(treeEntries.get(dir)??[])].sort((a,b)=>Number(b.directory)-Number(a.directory)||natural.compare(a.name,b.name));
  let cursor:ChildNode|null=container.firstChild;
  for(const e of items){
   seen.add(e.path);let n=fileNodes.get(e.path);
   if(!n){
    const wrap=node("div","","tree-node"),row=node("div","","tree-row");
    const b=node("button","","tree-item") as HTMLButtonElement;b.type="button";
    const arrow=node("span","","tree-arrow"),icon=node("span","","tree-icon"),name=node("span","","tree-name");
    arrow.setAttribute("aria-hidden","true");icon.setAttribute("aria-hidden","true");
    b.append(arrow,icon,name);const more=node("button","⋯","quiet tree-more") as HTMLButtonElement;more.type="button";
    const child=node("div","","branch"),judge=node("span","","tree-judge muted note");judge.title="远程评测缓存；AC 表示曾通过，不代表当前源码";row.append(b,judge,more);wrap.append(row,child);
    n={wrap,row,button:b,arrow,icon,name,more,child,judge};fileNodes.set(e.path,n)
   }
   const {wrap,row,button:b,arrow,icon,name,more,child,judge}=n;
   const isExpanded=e.directory&&expanded.has(e.path),isSelected=!e.directory&&source?.path===e.path;
   row.classList.toggle("selected",isSelected);b.setAttribute("aria-current",String(isSelected));b.title=e.path;b.disabled=busy;
   if(e.directory)b.setAttribute("aria-expanded",String(isExpanded));else b.removeAttribute("aria-expanded");
   arrow.classList.toggle("expanded",isExpanded);arrow.classList.toggle("leaf",!e.directory);
   icon.className="tree-icon "+(e.directory?"folder-icon":"source-icon");name.textContent=e.name;judge.textContent="";judge.hidden=true;
   b.onclick=()=>{if(e.directory){if(expanded.has(e.path)){expanded.delete(e.path);drawTree()}else{expanded.add(e.path);drawTree();void refreshTree(true)}}else if(!isSelected)void navigate(async()=>select(await call<Nav>("Select",e.path)))};
   more.hidden=!e.directory&&!isSelected;more.disabled=busy;more.title=e.directory?"文件夹操作":"当前文件操作";more.setAttribute("aria-label",more.title);
   more.onclick=()=>openFileMenu(more,e);
   b.oncontextmenu=ev=>{if(e.directory||isSelected){ev.preventDefault();if(!busy)openFileMenu(more,e)}};
   child.hidden=!isExpanded;if(isExpanded)branch(e.path,child,depth+1);
   if(wrap!==cursor)container.insertBefore(wrap,cursor);cursor=wrap.nextSibling;
  }
  while(cursor){const next=cursor.nextSibling;cursor.remove();cursor=next}
 }
 if(root)branch(root,host,0);else host.replaceChildren();
 for(const [path,n] of fileNodes)if(!seen.has(path)){n.wrap.remove();fileNodes.delete(path)}
 host.scrollTop=scroll;rememberExpansion();
}
async function locateFile(){
 if(!source)return;let dir=source.directory;
 while(dir&&dir!==root){expanded.add(dir);const next=dir.replace(/[\\/][^\\/]+$/,"");if(next===dir||next.length<root.length)break;dir=next}
 await refreshTree(true);drawTree();fileNodes.get(source.path)?.button.scrollIntoView({block:"nearest"});fileNodes.get(source.path)?.button.focus({preventScroll:true})
}
async function refreshTree(force=false){
 if(!root||refreshing||(!force&&busy)||(!force&&!fileMenu.hidden))return;
 refreshing=true;const epoch=navEpoch,base=root;
 try{
  const next=new Map<string,Entry[]>();const queue=[base];const present=new Set<string>();let scanned=0;
  while(queue.length&&scanned<128){const dir=queue.shift()!;scanned++;
   let list:Entry[];try{list=(await call<Entry[]|null>("List",dir))??[]}catch(e){if(dir===base)throw e;continue}
   if(epoch!==navEpoch)return;next.set(dir,list);
   for(const e of list){if(!e.directory)present.add(e.path);else if(expanded.has(e.path))queue.push(e.path)}
  }
  for(const [dir,entries] of next){const old=treeEntries.get(dir)??[];for(const e of old){if(!e.directory&&!present.has(e.path)){seenMissing.add(e.path);await call("Invalidate",e.path);if(epoch!==navEpoch)return}}}
  const changed=JSON.stringify([...next])!==JSON.stringify([...treeEntries]);treeEntries=next;if(changed||force)drawTree();if(changed&&workbench.active()&&!busy&&!force){workbench.invalidate();void syncOJWorkspace()}
  el("treeNotice").textContent=queue.length?"展开目录过多，最多刷新 128 个目录。":"";el("treeNotice").hidden=!el("treeNotice").textContent;
  if(source&&!busy&&!dirty&&!saveFlight){
   const selectedPath=source.path;const exists=await call<boolean>("CheckSelected");
   if(epoch!==navEpoch||source?.path!==selectedPath)return;
   if(!exists){seenMissing.add(selectedPath);markStale();loaded=false;updateControls();status.textContent="源文件已移走或删除；样例仍保留。"}
  }
  if(last&&source&&!dirty&&!busy&&!saveFlight){const id=last.runId;if(!(await call<boolean>("IsCurrent",compiler.value.trim()))&&last?.runId===id&&epoch===navEpoch)markStale()}
 }catch(e){el("treeNotice").textContent=String(e);el("treeNotice").hidden=false;markStale()}finally{refreshing=false}
}
async function storeConfig(){try{await call("StoreConfig",compilerByLanguage.c.trim(),compilerByLanguage.cpp.trim(),editorPath.trim())}catch(e){status.textContent="配置保存失败："+String(e)}}
async function navigate(action:()=>Promise<void>){
 if(busy||dialogs.isOpen())return;
 busy=true;updateControls();drawTree();try{await save();navEpoch++;await action();await storeConfig()}catch(e){status.textContent=String(e)}finally{busy=false;updateControls();drawTree();void refreshTree(true)}
}
async function save(){clearTimeout(saveTimer);if(saveFlight)await saveFlight;if(!dirty)return;if(!loaded)throw Error("样例未加载，无法保存");const generation=editGeneration,path=source?.path,rev=revision,data=cases.map(c=>({...c}));saveError="";const task=(async()=>{try{const set=await call<CaseSet>("SaveCases",data,rev);if(source?.path!==path)throw Error("保存目标已变化");revision=set.revision;if(generation===editGeneration)dirty=false;else queueSave();saveError=""}catch(e){saveError=String(e);throw e}})();saveFlight=task;updateControls();try{await task}finally{if(saveFlight===task)saveFlight=undefined;updateControls()}}
function node(tag:string,text:string,cls=""){const n=document.createElement(tag);n.textContent=text;if(cls)n.className=cls;return n;}
function output(title:string,text:string){const d=node("div","","output-block");if(title)d.append(node("h3",title));d.append(node("pre",text||"（空输出）"));return d}
function normalized(s:string){return s.replace(/\r\n/g,"\n").split("\n").map(l=>l.replace(/[ \t]+$/,"")).join("\n").replace(/\n+$/,"")}
function diffCell(text:string|undefined,other:string|undefined){
 const cell=node("td","");if(text===undefined){cell.textContent="—";cell.className="muted";return cell}
 const limit=4000;
 if(text.length>limit||(other?.length??0)>limit){cell.textContent=text.slice(0,limit)+(text.length>limit?"…":"");if(text!==other)cell.className="diffmark";return cell}
 if(text===other){cell.textContent=text||" ";return cell}
 const a=Array.from(text),b=Array.from(other??"");let p=0,s=0;
 while(p<a.length&&p<b.length&&a[p]===b[p])p++;
 while(s<a.length-p&&s<b.length-p&&a[a.length-1-s]===b[b.length-1-s])s++;
 cell.append(document.createTextNode(a.slice(0,p).join("")),node("span",a.slice(p,a.length-s).join("")||"∅","diffmark"),document.createTextNode(a.slice(a.length-s).join("")));
 return cell
}
function comparison(c:CaseResult){
 const e=normalized(c.expected??"").split("\n"),a=normalized(c.process.stdout).split("\n");
 const count=Math.max(e.length,a.length),table=document.createElement("table");table.className="output-table";
 const head=document.createElement("thead"),hr=document.createElement("tr");for(const label of ["行","预期","实际"])hr.append(node("th",label));head.append(hr);table.append(head);
 const body=document.createElement("tbody"),cap=Math.min(count,500);let clipped=false;
 // Preview around first known difference for long output. Comparison status
 // remains authoritative from Go; this view never recomputes pass/fail.
 const start=Math.max(0,Math.min(count-cap,(c.difference?.line??1)-20));
 for(let n=start;n<start+cap;n++){const row=document.createElement("tr"),x=e[n],y=a[n];if(x!==y)row.className="bad-row";if((x?.length??0)>4000||(y?.length??0)>4000)clipped=true;
 row.append(node("th",String(n+1)),diffCell(x,y),diffCell(y,x));body.append(row)}
 table.append(body);const wrap=node("div","","comparison");wrap.append(table);
 if(count>cap||clipped)wrap.append(node("p","对比仅展示部分内容；完整输出见原始文本。","muted note"));
 return wrap
}
function renderResult(){
 result.hidden=false;const wrap=el("results"),diag=el("diagnostics");wrap.replaceChildren();diag.replaceChildren();
 el("diagnosticsSection").hidden=true;btn("rawToggle").hidden=true;el("batchSummary").textContent="";
 const current=cases[selected];const candidate=caseResult(selected);const tableMode=!!(last&&candidate&&candidate.difference&&candidate.expected!==null&&!last.stale&&matches(candidate,selected)&&!editingExpected);el("outputLayout").classList.toggle("comparison-mode",tableMode||!current);el("outputHeading").textContent=tableMode?"输出对比":"实际输出";el("expectedEditor").hidden=!current||tableMode;btn("editExpected").hidden=!tableMode;stale.hidden=true;
 if(!last){el("runDetails").hidden=true;el("resultTitle").textContent="";el("resultTitle").className="";el("resultMeta").textContent="";wrap.append(node("p","尚未运行","muted"));return}el("runDetails").hidden=false;const r=last,c=caseResult(selected),past=!!c&&(r.stale||!matches(c,selected));
 stale.hidden=!r.stale&&!past;
 el("meta").textContent=r.compiler+(r.kitVersion?" · Exam Kit "+r.kitVersion+" · "+(r.components?.length??0)+" 个组件":"");
 const compared=r.cases.filter(c=>c.expected!==null),passed=compared.filter(c=>c.state==="passed").length;
 el("batchSummary").textContent=(r.stale?"上次 · ":"")+(r.state==="completed"?(compared.length?passed+"/"+compared.length+" 组通过":"已运行 "+r.cases.length+" 组"):labels[r.state]??r.state);
 el("resultTitle").textContent=c?labels[c.state]??c.state: r.state==="completed"?"当前样例尚未运行":labels[r.state]??r.state;
 el("resultTitle").className=c&&!past?c.state:"";
 el("resultMeta").textContent=c?c.process.elapsedMs+" ms"+(c.process.exitCode!==0?" · 退出码 "+c.process.exitCode:""):"";
 if(c){
  if(past){const frozen=document.createElement("details");frozen.className="snapshot";frozen.append(node("summary","历史运行快照（非当前样例）"),output("本次输入",c.input));if(c.expected!==null)frozen.append(output("本次预期",c.expected));wrap.append(frozen)}
  if(tableMode&&c.expected!==null&&c.difference){
   btn("rawToggle").hidden=false;btn("rawToggle").textContent=rawResult?"逐行对比":"原始文本";
   if(rawResult){const row=node("div","","outputs");row.append(output("预期",c.expected),output("实际",c.process.stdout));wrap.append(row)}
   else{wrap.append(comparison(c));wrap.append(node("p","首处差异 · 第 "+c.difference.line+" 行，第 "+c.difference.column+" 列","muted note"))}
  }else wrap.append(output(past?"上次实际输出":"",c.process.stdout));
  if(c.process.stderr)wrap.append(output("标准错误",c.process.stderr));
 }
 for(const x of r.diagnostics??[]){const d=node("div","","diagnostic");d.append(node("p",x.severity+" · "+(x.file?x.file.split(/[\\/]/).at(-1)+":"+x.line+"："+x.message:x.message)));const more=document.createElement("details");more.append(node("summary","完整诊断"),node("pre",x.raw));d.append(more);diag.append(d)}
 if(r.compilation.stdout)diag.append(output("编译输出",r.compilation.stdout));
 if(!diag.childElementCount&&r.compilation.stderr)diag.append(output("编译错误",r.compilation.stderr));
 el("diagnosticsSection").hidden=!diag.childElementCount;el("diagnosticSummary").textContent="编译诊断"+(r.diagnostics?.length?" · "+r.diagnostics.length+" 条":"");el<HTMLDetailsElement>("diagnosticsSection").open=r.state==="compilation_failed"||(r.diagnostics??[]).some(x=>x.severity.toLowerCase().includes("error"));
}
function show(r:Result){last=r;rawResult=false;editingExpected=false;status.textContent="";renderTabs();renderResult()}
btn("rawToggle").onclick=()=>{rawResult=!rawResult;renderResult()};btn("editExpected").onclick=()=>{editingExpected=true;renderResult();expected.focus()};
const fileMenu=el("fileMenu");let menuAnchor:HTMLElement|undefined,menuEntry:Entry|undefined;
function closeFileMenu(){fileMenu.hidden=true;menuAnchor=undefined}
function openFileMenu(anchor:HTMLElement,entry:Entry){if(!fileMenu.hidden){closeFileMenu();return}menuAnchor=anchor;menuEntry=entry;btn("rename").hidden=entry.directory;btn("kitFileMenu").hidden=!entry.directory&&!/\.c$/i.test(entry.path);btn("reveal").textContent=entry.directory?"打开文件夹":"打开所在目录";fileMenu.hidden=false;const box=anchor.getBoundingClientRect();fileMenu.style.left=Math.max(4,Math.min(box.right-fileMenu.offsetWidth,innerWidth-fileMenu.offsetWidth-4))+"px";fileMenu.style.top=Math.max(4,Math.min(box.bottom,innerHeight-fileMenu.offsetHeight-4))+"px";btn(entry.directory?"newHere":"rename").focus()}
document.addEventListener("pointerdown",e=>{if(!fileMenu.hidden&&!fileMenu.contains(e.target as Node)&&!menuAnchor?.contains(e.target as Node))closeFileMenu();for(const d of Array.from(document.querySelectorAll<HTMLDetailsElement>("details.menu[open]")))if(!d.contains(e.target as Node))d.open=false});
document.addEventListener("keydown",e=>{if(e.key==="Escape"&&!fileMenu.hidden){const a=menuAnchor;closeFileMenu();a?.focus()}if(e.key==="Escape")for(const d of Array.from(document.querySelectorAll<HTMLDetailsElement>("details.menu[open]")))d.open=false});
fileMenu.addEventListener("click",()=>closeFileMenu());
async function manageKit(path:string,group:string[]=[]){if(busy)return;busy=true;updateControls();try{await save();const r=await kitSwitch.run(path,group);if(r){if(source&&r.changed.includes(source.path)){last=undefined;prepared=undefined;renderCases();renderResult()}status.textContent="已切换 "+r.changed.length+" 个文件"+(r.failed.length?"；失败："+r.failed.join("；"):"")}}catch(e){status.textContent=String(e)}finally{busy=false;updateControls()}}
btn("kitCurrent").onclick=()=>{if(source)void manageKit(source.path)};btn("kitRoot").onclick=()=>void manageKit(root);btn("kitContest").onclick=()=>void manageKit("",workbench.groupPaths());btn("kitFileMenu").onclick=()=>{if(menuEntry)void manageKit(menuEntry.path)};
for(const d of Array.from(document.querySelectorAll<HTMLDetailsElement>("details.menu")))d.addEventListener("click",e=>{if((e.target as HTMLElement).closest("button"))d.open=false});
el("file").addEventListener("scroll",closeFileMenu);
window.addEventListener("resize",closeFileMenu);
const grip=el("sidebarGrip"),shell=el("shell");
let sidebarWidth=220;
function resizeSidebar(width:number){sidebarWidth=Math.max(150,Math.min(420,width));shell.style.setProperty("--sidebar-width",sidebarWidth+"px");grip.setAttribute("aria-valuenow",String(sidebarWidth));closeFileMenu()}
grip.addEventListener("pointerdown",e=>{grip.setPointerCapture(e.pointerId);e.preventDefault()});
grip.addEventListener("pointermove",e=>{if(grip.hasPointerCapture(e.pointerId))resizeSidebar(e.clientX-shell.getBoundingClientRect().left)});
grip.addEventListener("keydown",e=>{if(e.key==="ArrowLeft"||e.key==="ArrowRight"){e.preventDefault();resizeSidebar(sidebarWidth+(e.key==="ArrowLeft"?-12:12))}});
async function run(all:boolean){
 if(busy||!source||!loaded)return;
 busy=true;updateControls();renderCases();drawTree();
 try{await save();await storeConfig();runningTest=true;updateControls();el("progress").hidden=false;el("batchSummary").textContent="正在编译与运行…";status.textContent="";const r=await call<Result>("RunBatch",compiler.value.trim(),revision,all?-1:selected);show(r)}
 catch(e){status.textContent=String(e)}finally{runningTest=false;el("progress").hidden=true;busy=false;renderCases();drawTree()}
}
btn("open").onclick=()=>void navigate(async()=>select(await call<Nav>("ChooseSource")));
btn("openFolder").onclick=()=>void navigate(async()=>select(await call<Nav>("ChooseFolder")));
btn("refreshTree").onclick=()=>void refreshTree(true);
btn("ojImport").onclick=()=>{void navigate(async()=>{if(!await ojAccount.requireLogin())return;const r=await ojImport.run(source?.path??"",root);if(!r)return;await select(r.nav);await syncOJWorkspace();status.textContent=`导入完成 ${r.completed.length} 题，跳过 ${r.skipped.length} 题，失败 ${r.failed.length} 题。`+(r.failed.length?" "+r.failed.join("；"):"");const warning=await call<string>("KitCreationWarning");if(warning)status.textContent+="；新建 C 文件未接入 Exam Kit："+warning})};
btn("ojOpenOriginal").onclick=async()=>{const path=source?.path;if(!path||busy)return;try{await call("OJOpenOriginal",path)}catch(e){status.textContent=String(e)}};
btn("ojUnbind").onclick=async()=>{const path=source?.path;if(!path||busy||!await dialogs.confirm("解除题目关联？源码和样例不变。","解除关联")||source?.path!==path)return;busy=true;updateControls();try{await call("OJUnbind",path);await syncOJWorkspace();status.textContent="已解除题目关联；源码和样例保留。"}catch(e){status.textContent=String(e)}finally{busy=false;updateControls()}};
function newFile(dir=root){return navigate(async()=>{const target=await dialogs.create(dir,()=>call<string>("ChooseCreateDirectory"));if(target){await select(await call<Nav>("Create",target.dir,target.name));if(/\.c$/i.test(target.name)){const warning=await call<string>("KitCreationWarning");if(warning)status.textContent="已创建普通 C 文件；Exam Kit 未接入："+warning}}})}
btn("new").onclick=()=>void newFile();
btn("newHere").onclick=()=>{const e=menuEntry;void newFile(e?.directory?e.path:source?.directory??root)};
async function newFolder(dir:string){if(busy)return;busy=true;updateControls();drawTree();try{const name=await dialogs.name("新建文件夹","在 "+dir+" 中新建文件夹","新文件夹","新建");if(!name)return;const path=await call<string>("CreateFolder",dir,name);expanded.add(dir);expanded.add(path)}catch(e){status.textContent=String(e)}finally{busy=false;updateControls();await refreshTree(true);drawTree()}}
btn("newFolderHere").onclick=()=>{const e=menuEntry;void newFolder(e?.directory?e.path:source?.directory??root)};
btn("newRootFolder").onclick=()=>void newFolder(root);
btn("locateFile").onclick=()=>void locateFile();btn("collapseTree").onclick=()=>{expanded=new Set([root]);drawTree()};
btn("rename").onclick=()=>void navigate(async()=>{if(!source)return;const name=await dialogs.name("重命名文件","新文件名",source.name,"重命名");if(name&&name!==source.name)await select(await call<Nav>("Rename",name))});
btn("import").onclick=()=>void navigate(async()=>{if(!loaded)return;const append=cases.length>0;if(append&&!await dialogs.confirm("将导入的测试数据追加到现有样例？"))return;const s=await call<CaseSet|null>("Import",revision,append);if(s){applySet(s);markStale();await showOrphans()}});
setInterval(()=>{void refreshTree()},1500);
setInterval(async()=>{
 if(!runtime)return;
 if(runningTest&&!progressCheck){progressCheck=true;try{const p=await call<Progress>("Progress");if(!runningTest)return;const bar=el<HTMLProgressElement>("progress");if(p.phase==="compiling"){bar.removeAttribute("value");el("batchSummary").textContent="正在编译…"}else if(p.phase==="running"){bar.max=Math.max(1,p.total);bar.value=p.completed;el("batchSummary").textContent="正在运行 "+p.completed+" / "+p.total+" 组"}}catch{}finally{progressCheck=false}}
 if(dropCheck)return;dropCheck=true;
 try{const p=await call<string[]|null>("TakeDrop");if(!p?.length)return;if(p.length!==1){status.textContent="一次仅支持拖入一个文件或文件夹";return}if(busy||dialogs.isOpen()||el<HTMLDialogElement>("preview").open||el<HTMLDialogElement>("config").open||el<HTMLDialogElement>("manualDialog").open||el<HTMLDialogElement>("editorDialog").open||el<HTMLDialogElement>("ojAccountDialog").open||el<HTMLDialogElement>("kitDialog").open){status.textContent="请结束当前操作后重新拖入";return}await navigate(async()=>select(await call<Nav>("OpenDropped",p[0])))}catch(e){status.textContent=String(e)}finally{dropCheck=false}
},300);
btn("observe").onclick=async()=>{
 if(busy||observing||!source||!loaded)return;
 busy=true;updateControls();el("observerState").textContent="正在启动 Observer，检查当前源码与样例…";try{await save();await call("Observe",compiler.value.trim(),revision,selected);observing=true;el("observerState").textContent="Observer 正在运行；关闭观察窗口后清理会话。"}catch(e){el("observerState").textContent=String(e)}finally{busy=false;updateControls()}
};
setInterval(async()=>{
 if(!observing||observerPolling)return;observerPolling=true;
 try{const s=await call<string>("ObservationState");if(s!=="running"){observing=false;el("observerState").textContent=s==="closed"?"观察窗口已关闭，会话已清理。":s;updateControls()}}catch(e){el("observerState").textContent=String(e)}finally{observerPolling=false}
},1000);
function showObserver(o:ObserverInfo){el("observerConfigStatus").textContent=o.error?"不可用："+o.error:o.version||"不可用"}
const preview=el<HTMLDialogElement>("preview"),config=el<HTMLDialogElement>("config");
// Only traverse currently expanded branches; cached collapsed directories are not online refresh targets.
const ojAccount=installOJAccount(call,el<HTMLDialogElement>("ojAccountDialog"),()=>{void syncOJVisibility()});const kitSwitch=installKitSwitch(call);
const ojImport=installOJImport<Nav>(call);
const ojStatus=installOJStatus(call,()=>workbench.paths(),()=>source?.path??"",()=>workbench.draw(),message=>{status.textContent=message;if(preview.open&&!el("ojSubmitArea").hidden&&message.startsWith("提交 #"))el("ojSubmitNotice").textContent=message},()=>!busy&&!config.open);
const ojSubmit=installOJSubmit(call,preview,message=>{status.textContent=message},ojStatus.poll);
const documentReader=installDocument(call);
const workbench=installWorkbench(call,path=>{if(path!==source?.path)void navigate(async()=>select(await call<Nav>("Select",path)))},()=>{void syncOJWorkspace();if(workbench.active())void ojAccount.enter()},()=>source?.path??"",ojStatus.mark,()=>busy);
let workspaceEpoch=0;
async function syncOJWorkspace(rescan=true){const n=++workspaceEpoch;if(!workbench.active()){ojStatus.clear();return}if(ojAccount.connecting())return;if(rescan||workbench.needsScan())await workbench.scan();else workbench.draw();if(n!==workspaceEpoch||!workbench.active())return;await ojImport.refresh(source?.path??"");if(n!==workspaceEpoch||!workbench.active())return;btn("ojSubmitBtn").hidden=el("ojBinding").hidden;await ojStatus.refreshLocal();updateControls()}
const manual=installManual(call,()=>!!runtime&&!busy&&!dialogs.isOpen()&&!preview.open&&!config.open);
type EditorOption={name:string;path:string};
const editorDialog=el<HTMLDialogElement>("editorDialog");
let editorChoices:EditorOption[]=[];let editorEpoch=0;
function editorLabel(){return editorPath==="@system"?"系统默认编辑器":editorChoices.find(e=>e.path===editorPath)?.name??editorPath.split(/[\\/]/).at(-1)?.replace(/\.(exe|app)$/i,"")??""}
function updateEditorLabel(){btn("openEditor").textContent="编辑";btn("openEditor").title=editorPath?"打开 "+editorLabel():"打开编辑器";btn("settingsEditor").textContent=editorPath?editorLabel():"自动检测"}
async function launchEditor(path:string){try{await call("OpenEditor",path);editorPath=path;updateEditorLabel();await storeConfig();status.textContent="已请求打开 "+editorLabel();return true}catch(e){status.textContent=String(e);return false}}
async function editorMenu(openAfter:boolean){
 if(busy||(openAfter&&!source)||editorDialog.open)return;
 const epoch=++editorEpoch;editorDialog.showModal();const choices=el("editorChoices"),hint=el("editorHint");choices.replaceChildren();hint.textContent="正在检测本机代码编辑器…";
 async function pick(path:string){editorDialog.close();if(openAfter)await launchEditor(path);else{await preferenceFlight;try{await call("StoreConfig",compilerByLanguage.c.trim(),compilerByLanguage.cpp.trim(),path.trim());editorPath=path;updateEditorLabel();el("configNotice").hidden=true}catch(e){el("configNotice").textContent="编辑器设置保存失败："+String(e);el("configNotice").hidden=false}}}
 btn("editorClose").onclick=()=>editorDialog.close();
 btn("editorSystem").onclick=()=>void pick("@system");
 btn("editorCustom").onclick=async()=>{try{const p=await call<string>("ChooseEditor");if(p&&editorDialog.open&&epoch===editorEpoch)await pick(p)}catch(e){hint.textContent=String(e)}};
 try{const detected=await call<EditorOption[]>("DetectEditors")??[];if(!editorDialog.open||epoch!==editorEpoch)return;editorChoices=detected;
 hint.textContent=editorChoices.length?"选择代码编辑器，之后可以随时切换。":"未检测到代码编辑器，请手动选择。";
 for(const e of editorChoices){const b=node("button",e.name+(e.path===editorPath?" ✓":""),"editor-choice") as HTMLButtonElement;b.title=e.path;b.onclick=()=>void pick(e.path);choices.append(b)}
 if(openAfter&&!editorPath&&editorChoices.length===1)await pick(editorChoices[0].path);
 }catch(e){hint.textContent=String(e)}
}
btn("openEditor").onclick=async()=>{if(!editorPath){await editorMenu(true);return}if(!await launchEditor(editorPath))await editorMenu(true)};
btn("settingsEditor").onclick=()=>void editorMenu(false);
btn("reveal").onclick=async()=>{try{if(menuEntry?.directory)await call("RevealFolder",menuEntry.path);else await call("Reveal")}catch(e){status.textContent=String(e)}};
async function openSubmission(online:boolean){if(busy)return;busy=true;updateControls();try{prepared=await call<Submission>("Prepare",compiler.value.trim());el<HTMLTextAreaElement>("sourcePreview").value=prepared.text;el("verify").textContent=prepared.verified&&!dirty?"当前源码与全部已通过比较的样例一致。":"当前源码未完整验证（或有未保存样例）；请核对后再复制或提交。";if(prepared.kitVersion)el("verify").textContent+=" Exam Kit "+prepared.kitVersion+" · 展开 "+(prepared.components?.length??0)+" 个组件。";el("previewTitle").textContent="复制提交源码";preview.showModal();void ojSubmit.open(compiler.value.trim(),prepared.digest,online,source&&language(source)==="cpp"?"C++":"C")}catch(e){status.textContent=String(e)}finally{busy=false;updateControls()}}
btn("previewSource").onclick=()=>void openSubmission(false);btn("copy").onclick=async()=>{if(busy||!source)return;busy=true;updateControls();try{const p=await call<Submission>("Prepare",compiler.value.trim());await call("CopyPrepared",compiler.value.trim(),p.digest);status.textContent=p.verified&&!dirty?"已复制":"已复制；当前源码未完整验证"}catch(e){status.textContent=String(e)}finally{busy=false;updateControls()}};btn("ojSubmitBtn").onclick=async()=>{if(busy||!source)return;const path=source.path;if(!await ojAccount.requireLogin()||busy||source?.path!==path||!workbench.active())return;void openSubmission(true)};
btn("closePreview").onclick=()=>{prepared=undefined;preview.close()};
preview.onclose=()=>{prepared=undefined};
btn("copyConfirm").onclick=async()=>{if(!prepared)return;btn("copyConfirm").disabled=true;try{await call("CopyPrepared",compiler.value.trim(),prepared.digest);preview.close();status.textContent="已复制当前提交源码"}catch(e){preview.close();status.textContent=String(e)}finally{btn("copyConfirm").disabled=false}};
function showKit(k:KitStatus){el("kitStatus").textContent=k.error?"不可用："+k.error:k.version}
btn("settings").onclick=()=>{el<HTMLInputElement>("cc").value=compilerByLanguage.c;el<HTMLInputElement>("cxx").value=compilerByLanguage.cpp;config.showModal();el("configNotice").hidden=true;void ojAccount.status(true).then(syncOJVisibility);void call<ObserverInfo>("ObserverStatus").then(showObserver).catch(e=>{el("observerConfigStatus").textContent=String(e)});void call<KitStatus>("KitStatus").then(showKit).catch(e=>{el("kitStatus").textContent=String(e)})};
btn("closeConfig").onclick=async()=>{await savePreferences();if(el("configNotice").hidden)config.close()};
let lastAccountKey="";async function syncOJVisibility(){try{const a=await call<Parameters<typeof ojAccount.set>[0]>("OJAccountStatus");ojAccount.set(a);const key=a.available?(a.id??"")+":"+(a.state??""):"unavailable";if(key!==lastAccountKey){lastAccountKey=key;ojStatus.clear()}workbench.availability(a.available)}catch{ojStatus.clear();workbench.availability(false)}await syncOJWorkspace();updateControls()}
config.addEventListener("close",()=>{void syncOJVisibility()});
async function refreshOJ(paths:string[]){if(busy||!paths.length)return;busy=true;updateControls();try{if(await ojAccount.requireLogin())await ojStatus.refresh(paths)}finally{busy=false;updateControls()}}
btn("ojRefreshTree").onclick=()=>{void refreshOJ(workbench.groupPaths())};
btn("ojRefreshCurrent").onclick=()=>{if(source)void refreshOJ([source.path])};

let preferenceFlight:Promise<void>=Promise.resolve();function savePreferences(){const c=el<HTMLInputElement>("cc").value.trim(),cpp=el<HTMLInputElement>("cxx").value.trim();preferenceFlight=preferenceFlight.then(async()=>{if(c===compilerByLanguage.c&&cpp===compilerByLanguage.cpp)return;try{await call("StoreConfig",c,cpp,editorPath.trim());compilerByLanguage.c=c;compilerByLanguage.cpp=cpp;if(source)compiler.value=compilerByLanguage[language(source)];markStale();el("configNotice").hidden=true}catch(e){el("configNotice").textContent="保存失败："+String(e);el("configNotice").hidden=false}});return preferenceFlight}el<HTMLInputElement>("cc").onchange=()=>void savePreferences();el<HTMLInputElement>("cxx").onchange=()=>void savePreferences();
compiler.onchange=()=>void storeConfig();
btn("reload").onclick=()=>void navigate(async()=>{markStale();if(source&&seenMissing.has(source.path))await call("Invalidate",source.path);await reload();if(source&&seenMissing.has(source.path)&&revision!=="missing"){if(!await dialogs.confirm("此路径曾消失，是否继续使用旧样例？")){loaded=false;return}seenMissing.delete(source.path)}await showOrphans()});
btn("add").onclick=()=>{cases.push({input:"",expected:null});selected=cases.length-1;changes();renderCases()};
btn("remove").onclick=async()=>{if(busy||dialogs.isOpen())return;busy=true;updateControls();try{if(!await dialogs.confirm("删除当前样例？修改将自动保存。","删除"))return;cases.splice(selected,1);selected=Math.max(0,Math.min(selected,cases.length-1));changes()}catch(e){status.textContent=String(e)}finally{busy=false;renderCases();if(dirty)queueSave()}};
input.oninput=()=>{if(cases[selected]){cases[selected].input=input.value;changes()}};
expected.oninput=()=>{if(cases[selected]){cases[selected].expected=expected.value;changes()}};
compare.onchange=()=>{if(cases[selected]){cases[selected].expected=compare.checked?expected.value:null;changes();renderCases()}};
compiler.oninput=()=>{if(source)compilerByLanguage[language(source)]=compiler.value;markStale()};
btn("save").onclick=async()=>{busy=true;updateControls();try{await save()}catch(e){status.textContent=String(e)}finally{busy=false;renderCases()}};
btn("run").onclick=()=>void run(false);btn("all").onclick=()=>void run(true);
btn("cancel").onclick=async()=>{btn("cancel").disabled=true;try{await call("Cancel")}catch(e){status.textContent=String(e)}};
window.addEventListener("focus",async()=>{if(last&&!busy&&!dirty){const id=last.runId;try{if(!(await call<boolean>("IsCurrent",compiler.value.trim()))&&last?.runId===id)markStale()}catch{markStale()}}});
document.addEventListener("keydown",e=>{if((e.ctrlKey||e.metaKey)&&e.key==="Enter"&&!busy&&!dialogs.isOpen()&&source&&!preview.open&&!config.open&&!editorDialog.open&&!el<HTMLDialogElement>("ojAccountDialog").open&&!el<HTMLDialogElement>("kitDialog").open&&!manual.isOpen()&&!documentReader.isOpen()&&!(el<HTMLDialogElement>("ojDetailDialog").open)){e.preventDefault();void run(true)}if((e.ctrlKey||e.metaKey)&&e.key.toLowerCase()==="h"&&!dialogs.isOpen()&&!manual.isOpen()&&!preview.open&&!config.open&&!editorDialog.open&&!el<HTMLDialogElement>("ojAccountDialog").open&&!el<HTMLDialogElement>("kitDialog").open&&!documentReader.isOpen()&&!el<HTMLDialogElement>("ojDetailDialog").open&&!busy){e.preventDefault();void manual.open()}});
window.addEventListener("beforeunload",e=>{if(dirty||saveFlight){e.preventDefault();e.returnValue=""}});
async function boot(){try{const path="/wails/runtime.js";runtime=await import(path) as Runtime;if(typeof runtime.Call?.ByName!=="function")throw Error("运行时不可用");el("appVersion").textContent="v"+await call<string>("Version");const cfg=await call<Settings>("Config");compilerByLanguage.c=cfg.c;compilerByLanguage.cpp=cfg.cpp;editorPath=cfg.editor;updateEditorLabel();status.textContent="";updateControls();await select(await call<Nav>("Initial"));const warning=await call<string>("ConfigWarning");if(warning)status.textContent="配置读取失败："+warning;void call("OJCheckAdapter").then(syncOJVisibility).catch(()=>{})}catch(e){runtime=undefined;status.textContent="无法连接本地应用："+String(e);updateControls()}}
void boot();

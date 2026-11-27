type Call=<T>(method:string,...args:unknown[])=>Promise<T>;
type Item={path:string;directory:string;group:string;binding:{title:string;label:string;contestTitle?:string;target:{kind:string;contestId?:string;problemId:string}}};
type View={items:Item[];limited:boolean};
const el=<T extends HTMLElement=HTMLElement>(id:string)=>document.getElementById(id) as T;
const btn=(id:string)=>el<HTMLButtonElement>(id);
export function installWorkbench(call:Call,choose:(path:string)=>void,changed:()=>void,selected:()=>string,mark:(path:string)=>string,isBusy:()=>boolean){
 let active=false,available=false,rows:Item[]=[],epoch=0,groupKey="",lastSelected="",scanned=false;const picker=el<HTMLSelectElement>("ojContestSelect");picker.onchange=()=>{groupKey=picker.value;draw()};
 const group=(v:Item)=>v.group+" · "+(v.binding.target.kind==="contest"?"比赛 #"+v.binding.target.contestId:"题库");
 function paths(){return active?rows.map(v=>v.path):[]}
 function groupPaths(){const current=rows.find(v=>v.path===selected());const k=groupKey||(current?group(current):"");return rows.filter(v=>group(v)===k).map(v=>v.path)}
 function draw(){
  if(!active)return;const host=el("ojTree"),scroll=host.scrollTop;host.replaceChildren();
  const groups=new Map<string,Item[]>();for(const item of rows){const k=group(item);if(!groups.has(k))groups.set(k,[]);groups.get(k)!.push(item)}
  const current=rows.find(v=>v.path===selected());if(current&&selected()!==lastSelected)groupKey=group(current);lastSelected=selected();if(!groups.has(groupKey))groupKey=groups.keys().next().value??"";
  picker.replaceChildren();picker.hidden=!groups.size;
  for(const [key,items] of groups){const option=new Option((items[0].binding.contestTitle||items[0].group)+" · "+items.filter(v=>mark(v.path)==="AC").length+"/"+items.length,key);picker.append(option)}picker.value=groupKey;
  const items=groups.get(groupKey)??[],matrix=items.length>0&&items.every(v=>v.binding.target.kind==="contest"&&/^[A-Z]{1,3}$/.test(v.binding.label))&&new Set(items.map(v=>v.binding.label)).size===items.length;
  host.classList.toggle("oj-matrix",matrix);
  for(const v of [...items].sort((a,b)=>a.binding.label.length-b.binding.label.length||a.binding.label.localeCompare(b.binding.label,undefined,{numeric:true})||a.path.localeCompare(b.path))){
    const row=document.createElement("button");row.type="button";row.className="oj-problem";row.setAttribute("aria-current",String(v.path===selected()));row.disabled=isBusy();row.title=v.binding.label+" · "+v.binding.title+" · "+v.path;const text=document.createElement("span");text.textContent=matrix?v.binding.label:v.binding.label+" · "+v.binding.title;const badge=document.createElement("small");badge.textContent=mark(v.path)||"—";badge.className=badge.textContent==="AC"?"passed":"";row.setAttribute("aria-label",v.binding.label+" · "+v.binding.title+" · "+(mark(v.path)||"未查询"));row.append(text,badge);row.onclick=()=>choose(v.path);host.append(row)
  }
  host.scrollTop=scroll;btn("ojRefreshTree").disabled=isBusy()||groupPaths().length===0
 }
 async function scan(){
  const n=++epoch;if(!active)return;
  try{const v=await call<View>("OJWorkspace");if(n!==epoch||!active)return;rows=v.items??[];scanned=true;el("ojWorkspaceNotice").textContent=v.limited?"仅展示扫描上限内的关联；可打开更小的目录。":rows.length?"":"当前目录尚无关联题目。"}catch(e){if(n!==epoch)return;rows=[];el("ojWorkspaceNotice").textContent=String(e)}draw()
 }
 function show(v:boolean){
  active=v&&available;epoch++;if(!active){rows=[];groupKey="";lastSelected="";scanned=false}
  btn("filesMode").setAttribute("aria-pressed",String(!active));btn("ojMode").setAttribute("aria-pressed",String(active));el("fileSidebar").hidden=active;el("ojSidebar").hidden=!active;btn("ojImport").hidden=!active;btn("ojRefreshTree").hidden=!active;
  if(!active){el("ojBinding").hidden=true;el("ojBindingError").hidden=true;el("ojJudgeInfo").hidden=true;btn("ojSubmitBtn").hidden=true}
  changed()
 }
 btn("filesMode").onclick=()=>{if(!isBusy())show(false)};
 btn("ojMode").onclick=()=>{if(!isBusy())show(true)};
 btn("ojRescan").onclick=()=>{if(!isBusy())changed()};
 function availability(v:boolean){available=v;btn("ojMode").hidden=!v;btn("filesMode").hidden=!v;if(active&&!v)show(false)}
 function controls(){if(!active)return;for(const b of Array.from(el("ojTree").querySelectorAll<HTMLButtonElement>("button")))b.disabled=isBusy();btn("ojRefreshTree").disabled=isBusy()||groupPaths().length===0;btn("ojRescan").disabled=isBusy();picker.disabled=isBusy()}
 return {active:()=>active,paths,groupPaths,scan,draw,availability,controls,needsScan:()=>!scanned,invalidate:()=>{scanned=false}}
}

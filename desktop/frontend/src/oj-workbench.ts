type Call=<T>(method:string,...args:unknown[])=>Promise<T>;
type Item={path:string;directory:string;group:string;binding:{title:string;label:string;contestTitle?:string;target:{kind:string;contestId?:string;problemId:string}}};
type View={items:Item[];limited:boolean};
const el=<T extends HTMLElement=HTMLElement>(id:string)=>document.getElementById(id) as T;
const btn=(id:string)=>el<HTMLButtonElement>(id);
export function installWorkbench(call:Call,choose:(path:string)=>void,changed:()=>void,selected:()=>string,mark:(path:string)=>string,isBusy:()=>boolean){
 let active=false,available=false,rows:Item[]=[],epoch=0,groupKey="",lastSelected="",lastDrawnGroup="",scanned=false;const picker=el<HTMLSelectElement>("ojContestSelect");picker.onchange=()=>{groupKey=picker.value;draw()};
 const group=(v:Item)=>v.group+" · "+(v.binding.target.kind==="contest"?"比赛 #"+v.binding.target.contestId:"题库");
 function paths(){return active?rows.map(v=>v.path):[]}
 function groupPaths(){const current=rows.find(v=>v.path===selected());const k=groupKey||(current?group(current):"");return rows.filter(v=>group(v)===k).map(v=>v.path)}
 function materialPaths(){const paths=groupPaths();return active&&paths.length&&paths.every(path=>rows.find(v=>v.path===path)?.binding.target.kind==="contest")?paths:[]}
 function materialControls(){btn("ojRefreshMaterials").hidden=!materialPaths().length;btn("ojRefreshMaterials").disabled=isBusy()||!materialPaths().length}
 function draw(){
  if(!active)return;const host=el("ojTree"),scroll=host.scrollTop,selectionChanged=selected()!==lastSelected,focusPath=(document.activeElement as HTMLElement|null)?.dataset.path;host.replaceChildren();
  const groups=new Map<string,Item[]>();for(const item of rows){const k=group(item);if(!groups.has(k))groups.set(k,[]);groups.get(k)!.push(item)}
  const current=rows.find(v=>v.path===selected());if(current&&selected()!==lastSelected)groupKey=group(current);lastSelected=selected();if(!groups.has(groupKey))groupKey=groups.keys().next().value??"";
  picker.replaceChildren();picker.hidden=!groups.size;
  for(const [key,items] of groups){const option=new Option((items[0].binding.contestTitle||items[0].group)+" · "+items.filter(v=>mark(v.path)==="AC").length+"/"+items.length,key);picker.append(option)}picker.value=groupKey;picker.title=picker.selectedOptions[0]?.textContent??"";
  const items=groups.get(groupKey)??[],matrix=items.length>0&&items.every(v=>v.binding.target.kind==="contest"&&/^[A-Z]{1,3}$/.test(v.binding.label))&&new Set(items.map(v=>v.binding.label)).size===items.length;
  host.classList.toggle("oj-matrix",matrix);const known=items.filter(v=>!!mark(v.path)).length,accepted=items.filter(v=>mark(v.path)==="AC").length;el("ojContestSummary").textContent=items.length?items.length+" 题 · "+(known?accepted+" 曾通过"+(known<items.length?" · "+known+" 已查询":""):"未查询"):"";const groupChanged=groupKey!==lastDrawnGroup;lastDrawnGroup=groupKey;let currentButton:HTMLButtonElement|undefined,focusedButton:HTMLButtonElement|undefined;
  for(const v of [...items].sort((a,b)=>a.binding.label.length-b.binding.label.length||a.binding.label.localeCompare(b.binding.label,undefined,{numeric:true})||a.path.localeCompare(b.path))){
    const row=document.createElement("button");row.type="button";row.className="oj-problem ui-button";row.setAttribute("aria-current",String(v.path===selected()));row.disabled=isBusy();row.dataset.path=v.path;const state=mark(v.path);row.title=v.binding.label+" · "+v.binding.title+" · "+(state==="AC"?"曾通过":state||"未查询")+"\n"+v.path;const text=document.createElement("span");text.textContent=matrix?v.binding.label:v.binding.label+" · "+v.binding.title;const badge=document.createElement("small");badge.textContent=matrix?(state==="AC"?"✓":state==="WT"||state==="JG"?"…":state&&state!=="未提交"?"×":""):(state==="未提交"?"":state);badge.className=state==="AC"?"passed":state&&state!=="未提交"&&state!=="WT"&&state!=="JG"?"judge-other":"";badge.hidden=!badge.textContent;row.setAttribute("aria-label",v.binding.label+" · "+v.binding.title+" · "+(state==="AC"?"曾通过":state||"未查询"));row.append(text,badge);row.onclick=()=>choose(v.path);host.append(row);if(v.path===selected())currentButton=row;if(v.path===focusPath)focusedButton=row
  }
  host.scrollTop=groupChanged?0:scroll;if((selectionChanged||groupChanged)&&currentButton){const view=host.getBoundingClientRect(),item=currentButton.getBoundingClientRect();if(item.top<view.top)host.scrollTop-=view.top-item.top;else if(item.bottom>view.bottom)host.scrollTop+=item.bottom-view.bottom}if(focusedButton&&!focusedButton.disabled)focusedButton.focus({preventScroll:true});btn("ojRefreshTree").disabled=isBusy()||groupPaths().length===0;materialControls()
 }
 async function scan(){
  const n=++epoch;if(!active)return;
  try{const v=await call<View>("OJWorkspace");if(n!==epoch||!active)return;rows=v.items??[];scanned=true;el("ojWorkspaceNotice").textContent=v.limited?"仅展示扫描上限内的关联；可打开更小的目录。":rows.length?"":"当前目录尚无关联题目。"}catch(e){if(n!==epoch)return;rows=[];el("ojWorkspaceNotice").textContent=String(e)}draw()
 }
 function show(v:boolean){
  active=v&&available;epoch++;if(!active){rows=[];groupKey="";lastSelected="";lastDrawnGroup="";scanned=false}
  btn("filesMode").setAttribute("aria-pressed",String(!active));btn("ojMode").setAttribute("aria-pressed",String(active));el("fileSidebar").hidden=active;el("ojSidebar").hidden=!active;btn("ojImport").hidden=!active;btn("ojRefreshTree").hidden=!active;materialControls();
  if(!active){el("ojSourceActions").hidden=true;el("ojBinding").hidden=true;el("ojBindingError").hidden=true;el("ojJudgeInfo").hidden=true;btn("ojSubmitBtn").hidden=true}
  changed()
 }
 btn("filesMode").onclick=()=>{if(!isBusy())show(false)};
 btn("ojMode").onclick=()=>{if(!isBusy())show(true)};
 btn("ojRescan").onclick=()=>{if(!isBusy())changed()};
 function availability(v:boolean){available=v;btn("ojMode").hidden=!v;btn("filesMode").hidden=!v;if(active&&!v)show(false)}
 function controls(){materialControls();if(!active)return;for(const b of Array.from(el("ojTree").querySelectorAll<HTMLButtonElement>("button")))b.disabled=isBusy();btn("ojRefreshTree").disabled=isBusy()||groupPaths().length===0;btn("ojRescan").disabled=isBusy();picker.disabled=isBusy()}
 return {active:()=>active,paths,groupPaths,materialPaths,scan,draw,availability,controls,needsScan:()=>!scanned,invalidate:()=>{scanned=false}}
}

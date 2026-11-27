type Call = <T>(method:string,...args:unknown[])=>Promise<T>;
type Row={name:string;title:string;action:"create"|"attach"|"repair"|"skip";samples:number;warnings:string[];document?:string};
type Preview={token:string;plan:{title:string;directory:string;rows:Row[];folderName?:string;canRename?:boolean}};
type Binding={title:string;label:string;sourceUrl:string;target:{kind:string}};
type Apply<T>={nav:T;completed:string[];skipped:string[];failed:string[]};
const el=<T extends HTMLElement=HTMLElement>(id:string)=>document.getElementById(id) as T;
const btn=(id:string)=>el<HTMLButtonElement>(id);
function modal(dialog:HTMLDialogElement):Promise<boolean>{return new Promise(resolve=>{dialog.returnValue="";dialog.addEventListener("close",()=>resolve(dialog.returnValue==="confirm"),{once:true});dialog.showModal()})}
export function installOJImport<T>(call:Call){
 const dialog=el<HTMLDialogElement>("ojImportDialog"),preview=el<HTMLDialogElement>("ojImportPreview");
 let epoch=0,importEpoch=0;
 async function refresh(path:string){
  const n=++epoch,binding=el("ojBinding");binding.hidden=true;el("ojBindingError").hidden=true;
  let available=false;try{available=(await call<{available:boolean}>("OJAccountStatus")).available}catch{/* unavailable */}
  if(n!==epoch)return false;
  if(!available||!path||btn("ojImport").hidden)return available;try{const b=await call<Binding|null>("OJBinding");if(n!==epoch||!b||btn("ojImport").hidden)return available;el("ojBindingTitle").textContent=`${b.label} · ${b.title}`;el("ojBindingTitle").title=b.title+" · "+path;btn("ojOpenOriginal").textContent=b.target.kind==="contest"?"打开原比赛 ↗":"打开原题 ↗";binding.hidden=false}catch{if(n===epoch&&!btn("ojImport").hidden){el("ojBindingError").textContent="题目关联无法读取，原文件保留。";el("ojBindingError").hidden=false}}
  return available;
 }
 async function run(path:string,root:string):Promise<Apply<T>|null>{
  const runEpoch=++importEpoch;
  const a=await call<{available:boolean;state:string}>("OJAccountStatus");if(!a.available)throw Error("Accoding Adapter 不可用");if(a.state!=="signed_in"&&a.state!=="unverified")throw Error("请先登录 Accoding");
  const url=el<HTMLInputElement>("ojImportURL"),lang=el<HTMLSelectElement>("ojImportLang"),attach=el<HTMLInputElement>("ojImportAttach");
  const recent=el("ojRecent"),next=btn("ojImportNext"),notice=el("ojRecentNotice");let mode:"recent"|"link"="recent",chosen="",destination=root;
  url.value="";attach.checked=false;lang.value=".c";recent.replaceChildren();notice.textContent="正在读取近期比赛…";el("ojDestination").textContent=destination;
  function draw(){const manual=mode==="link";btn("ojRecentMode").setAttribute("aria-pressed",String(!manual));btn("ojLinkMode").setAttribute("aria-pressed",String(manual));el("ojRecentPanel").hidden=manual;el("ojLinkPanel").hidden=!manual;el("ojAttachField").hidden=!manual||!path;url.required=manual;url.disabled=!manual;attach.disabled=!manual||!path;if(attach.disabled)attach.checked=false;btn("ojChooseDestination").disabled=attach.checked;next.disabled=manual?!url.value.trim()||!url.validity.valid:!chosen}
  btn("ojRecentMode").onclick=()=>{mode="recent";draw()};btn("ojLinkMode").onclick=()=>{mode="link";draw();url.focus()};url.oninput=draw;attach.onchange=()=>{if(attach.checked&&path)lang.value=/\.c$/i.test(path)?".c":".cpp";draw()};draw();
  btn("ojChooseDestination").onclick=async()=>{try{const p=await call<string>("ChooseCreateDirectory");if(p&&dialog.open&&runEpoch===importEpoch){destination=p;el("ojDestination").textContent=p}}catch(e){if(dialog.open&&runEpoch===importEpoch)notice.textContent=String(e)}};
  btn("ojMoreContests").onclick=()=>{void call("OJOpenContestList").catch(e=>{notice.textContent=String(e)})};
  const confirmed=modal(dialog);
  void call<{id:string;title:string;start:string;status:string}[]>("OJRecentContests").then(rows=>{
   if(!dialog.open||runEpoch!==importEpoch)return;
   for(const r of rows??[]){const row=document.createElement("button");row.type="button";row.className="recent-contest";row.setAttribute("aria-pressed","false");const title=document.createElement("span");title.textContent=r.title;const meta=document.createElement("small");meta.textContent=[r.start.slice(0,10),r.status].filter(Boolean).join(" · ");row.append(title,meta);row.onclick=()=>{chosen=r.id;for(const item of Array.from(recent.querySelectorAll("button")))item.setAttribute("aria-pressed",String(item===row));draw()};recent.append(row)}
   notice.textContent=rows?.length?"":"未找到近期比赛，可切换到输入链接";
  }).catch(()=>{if(dialog.open&&runEpoch===importEpoch)notice.textContent="比赛列表暂不可用，可切换到输入链接"});
  if(!await confirmed)return null;
  const link=mode==="recent"?(chosen?"https://accoding.buaa.edu.cn/contest-ng/index.html#/"+chosen:""):url.value.trim();if(!link)return null;
  let p=await call<Preview>("OJPreviewImportAt",link,lang.value,mode==="link"&&attach.checked,destination);const folder=el<HTMLInputElement>("ojFolderName"),field=el("ojFolderField"),error=el("ojFolderError");let replanBusy=false;field.hidden=!p.plan.canRename;folder.value=p.plan.folderName??"";folder.disabled=false;error.textContent="";el("ojPreviewTitle").textContent=p.plan.title;el("ojPreviewDir").textContent=p.plan.directory;
  function drawPlan(){const list=el("ojPreviewRows");list.replaceChildren();let count=0;for(const row of p.plan.rows){const item=document.createElement("li");const action=row.action==="create"?"新建":row.action==="attach"?"关联":row.action==="repair"?"补充":"跳过";if(row.action!=="skip")count++;item.textContent=`${action} ${row.name} · ${row.title} · ${row.samples} 组样例${row.document?" · "+row.document:""}${row.warnings.length?" · "+row.warnings.join("；"):""}`;list.append(item)}btn("ojPreviewConfirm").disabled=count===0||replanBusy||(!field.hidden&&folder.value!==p.plan.folderName)}drawPlan();
  folder.oninput=()=>{error.textContent="";drawPlan()};
  folder.onchange=async()=>{if(replanBusy||folder.value===p.plan.folderName)return;replanBusy=true;folder.disabled=true;btn("ojPreviewConfirm").disabled=true;try{const next=await call<Preview>("OJReplanImport",p.token,folder.value);if(preview.open&&runEpoch===importEpoch){p=next;folder.value=p.plan.folderName??"";field.hidden=!p.plan.canRename;error.textContent=""}}catch(e){if(preview.open&&runEpoch===importEpoch)error.textContent=String(e)}finally{replanBusy=false;if(runEpoch===importEpoch){folder.disabled=false;drawPlan()}}};
  if(!await modal(preview)||replanBusy||(!field.hidden&&folder.value!==p.plan.folderName))return null;return await call<Apply<T>>("OJApplyImport",p.token)
 }
 return {refresh,run};
}

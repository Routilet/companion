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
  if(!available||!path||btn("ojImport").hidden)return available;try{const b=await call<Binding|null>("OJBinding");if(n!==epoch||!b||btn("ojImport").hidden)return available;el("ojBindingTitle").textContent=`Accoding · ${b.label} · ${b.title}`;btn("ojOpenOriginal").textContent=b.target.kind==="contest"?"打开原比赛 ↗":"打开原题 ↗";binding.hidden=false}catch{if(n===epoch&&!btn("ojImport").hidden){el("ojBindingError").textContent="题目关联无法读取，原文件保留。";el("ojBindingError").hidden=false}}
  return available;
 }
 async function run(path:string,root:string):Promise<Apply<T>|null>{
  const runEpoch=++importEpoch;
  const a=await call<{available:boolean;state:string}>("OJAccountStatus");if(!a.available)throw Error("Accoding Adapter 不可用");if(a.state!=="signed_in"&&a.state!=="unverified")throw Error("请先在设置中登录 Accoding");
  const url=el<HTMLInputElement>("ojImportURL"),lang=el<HTMLSelectElement>("ojImportLang"),attach=el<HTMLInputElement>("ojImportAttach");url.value="";attach.checked=false;attach.disabled=!path;lang.value=".c";attach.onchange=()=>{btn("ojChooseDestination").disabled=attach.checked;if(attach.checked&&path)lang.value=/\.c$/i.test(path)?".c":".cpp"};btn("ojChooseDestination").disabled=false;
  let destination=root;const recent=el<HTMLSelectElement>("ojRecent");recent.replaceChildren(new Option("手动输入链接",""));el("ojDestination").textContent=destination;el("ojRecentNotice").textContent="正在读取近期比赛…";
  btn("ojChooseDestination").onclick=async()=>{try{const p=await call<string>("ChooseCreateDirectory");if(p&&dialog.open){destination=p;el("ojDestination").textContent=p}}catch(e){el("ojRecentNotice").textContent=String(e)}};
  recent.onchange=()=>{if(recent.value)url.value="https://accoding.buaa.edu.cn/contest-ng/index.html#/"+recent.value};url.oninput=()=>{recent.value=""};btn("ojMoreContests").onclick=()=>{void call("OJOpenContestList").catch(e=>{el("ojRecentNotice").textContent=String(e)})};
  const confirmed=modal(dialog);void call<{id:string;title:string;start:string;status:string}[]>("OJRecentContests").then(rows=>{if(!dialog.open)return;for(const r of rows??[])recent.append(new Option(r.start.slice(0,10)+" · "+r.title+" · "+r.status,r.id));el("ojRecentNotice").textContent=rows?.length?"":"未找到近期比赛，可手动输入链接"}).catch(()=>{if(dialog.open)el("ojRecentNotice").textContent="列表暂不可用，仍可手动输入链接"});
  if(!await confirmed)return null;let p=await call<Preview>("OJPreviewImportAt",url.value,lang.value,attach.checked,destination);const folder=el<HTMLInputElement>("ojFolderName"),field=el("ojFolderField"),error=el("ojFolderError");let replanBusy=false;field.hidden=!p.plan.canRename;folder.value=p.plan.folderName??"";folder.disabled=false;error.textContent="";el("ojPreviewTitle").textContent=p.plan.title;el("ojPreviewDir").textContent=p.plan.directory;
  function drawPlan(){const list=el("ojPreviewRows");list.replaceChildren();let count=0;for(const row of p.plan.rows){const item=document.createElement("li");const action=row.action==="create"?"新建":row.action==="attach"?"关联":row.action==="repair"?"补充":"跳过";if(row.action!=="skip")count++;item.textContent=`${action} ${row.name} · ${row.title} · ${row.samples} 组样例${row.document?" · "+row.document:""}${row.warnings.length?" · "+row.warnings.join("；"):""}`;list.append(item)}btn("ojPreviewConfirm").disabled=count===0||replanBusy||(!field.hidden&&folder.value!==p.plan.folderName)}drawPlan();
  folder.oninput=()=>{error.textContent="";drawPlan()};
  folder.onchange=async()=>{if(replanBusy||folder.value===p.plan.folderName)return;replanBusy=true;folder.disabled=true;btn("ojPreviewConfirm").disabled=true;try{const next=await call<Preview>("OJReplanImport",p.token,folder.value);if(preview.open&&runEpoch===importEpoch){p=next;folder.value=p.plan.folderName??"";field.hidden=!p.plan.canRename;error.textContent=""}}catch(e){if(preview.open&&runEpoch===importEpoch)error.textContent=String(e)}finally{replanBusy=false;if(runEpoch===importEpoch){folder.disabled=false;drawPlan()}}};
  if(!await modal(preview)||replanBusy||(!field.hidden&&folder.value!==p.plan.folderName))return null;return await call<Apply<T>>("OJApplyImport",p.token)
 }
 return {refresh,run};
}

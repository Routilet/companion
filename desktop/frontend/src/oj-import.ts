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
 // The backend does not expose per-problem progress. Use an indeterminate
 // indicator rather than guessing a percentage or offering a false cancel.
 const progressDialog=document.createElement("dialog");
 progressDialog.id="ojImportProgress";progressDialog.setAttribute("aria-labelledby","ojImportProgressText");
 progressDialog.style.width="min(360px,90vw)";
 const progressText=document.createElement("h2");progressText.id="ojImportProgressText";progressText.setAttribute("role","status");
 const progressBar=document.createElement("progress");progressBar.setAttribute("aria-label","导入进度（不确定）");
 progressDialog.append(progressText,progressBar);document.body.append(progressDialog);
 progressDialog.addEventListener("cancel",event=>event.preventDefault());
 async function withProgress<R>(message:string,action:()=>Promise<R>):Promise<R>{
  progressText.textContent=message;progressDialog.showModal();
  try{return await action()}finally{progressDialog.close()}
 }
 const recentProgress=document.createElement("progress");recentProgress.id="ojRecentProgress";recentProgress.setAttribute("aria-label","正在读取近期比赛");recentProgress.hidden=true;el("ojRecentNotice").after(recentProgress);
 function clearBinding(){el("ojBinding").hidden=true;el("ojSourceActions").hidden=true;el("ojBindingError").hidden=true;el("path").hidden=false;const file=el("path").title.split(/[\\/]/).pop();if(file&&/\.(c|cpp|cc|cxx)$/i.test(file)){el("title").textContent=file;el("title").title=el("path").title}}
 async function refresh(path:string){
  const n=++epoch,binding=el("ojBinding");clearBinding();
  let available=false;try{available=(await call<{available:boolean}>("OJAccountStatus")).available}catch{/* unavailable */}
  if(n!==epoch)return false;
  if(!available||!path||btn("ojImport").hidden)return available;try{const b=await call<Binding|null>("OJBinding");if(n!==epoch||!b||btn("ojImport").hidden)return available;el("title").textContent=`${b.label} · ${b.title}`;el("title").title=`${b.label} · ${b.title}\n${path}`;el("path").hidden=true;el("ojSourceActions").hidden=false;btn("ojOpenOriginal").textContent=b.target.kind==="contest"?"打开原比赛 ↗":"打开原题 ↗";binding.hidden=false}catch{if(n===epoch&&!btn("ojImport").hidden){el("ojBindingError").textContent="题目关联无法读取，原文件保留。";el("ojBindingError").hidden=false}}
  return available;
 }
 async function run(path:string,root:string):Promise<Apply<T>|null>{
  const runEpoch=++importEpoch;
  const a=await call<{available:boolean;state:string}>("OJAccountStatus");if(!a.available)throw Error("Accoding Adapter 不可用");if(a.state!=="signed_in"&&a.state!=="unverified")throw Error("请先登录 Accoding");
  const url=el<HTMLInputElement>("ojImportURL"),lang=el<HTMLSelectElement>("ojImportLang"),attach=el<HTMLInputElement>("ojImportAttach"),recent=el("ojRecent"),notice=el("ojRecentNotice"),confirm=btn("ojImportConfirm");
  let destination=root,manual=false,contestId="",loading=true;url.value="";attach.checked=false;attach.disabled=!path;lang.value=".c";recent.replaceChildren();el("ojDestination").textContent=destination;notice.textContent="正在读取近期比赛…";
  function update(){
   recentProgress.hidden=!loading||manual;el("ojRecentPanel").hidden=manual;el("ojLinkPanel").hidden=!manual;btn("ojImportMode").textContent=manual?"返回近期比赛":"通过链接导入";
   url.required=manual;confirm.disabled=manual?!url.value.trim()||!url.validity.valid:loading||!contestId;
   btn("ojChooseDestination").disabled=manual&&attach.checked;
  }
  function mode(v:boolean){manual=v;if(!v)attach.checked=false;update();if(v)url.focus()}
  btn("ojImportMode").onclick=()=>mode(!manual);
  attach.onchange=()=>{if(attach.checked&&path)lang.value=/\.c$/i.test(path)?".c":".cpp";update()};
  url.oninput=update;
  btn("ojChooseDestination").onclick=async()=>{try{const p=await call<string>("ChooseCreateDirectory");if(p&&dialog.open&&runEpoch===importEpoch){destination=p;el("ojDestination").textContent=p}}catch(e){if(dialog.open&&runEpoch===importEpoch)notice.textContent=String(e)}};
  btn("ojMoreContests").onclick=()=>{void call("OJOpenContestList").catch(e=>{notice.textContent=String(e)})};
  update();const confirmed=modal(dialog);
  void call<{id:string;title:string;start:string;status:string}[]>("OJRecentContests").then(rows=>{
   if(!dialog.open||runEpoch!==importEpoch)return;
   loading=false;
   for(const r of rows??[]){
    const button=document.createElement("button");button.type="button";button.className="recent-contest";button.setAttribute("aria-pressed","false");
    const name=document.createElement("span");name.textContent=r.title;const meta=document.createElement("small");meta.textContent=[r.start.slice(0,10),r.status].filter(Boolean).join(" · ");button.append(name,meta);
    button.onclick=()=>{contestId=r.id;recent.querySelectorAll("button").forEach(item=>item.setAttribute("aria-pressed",String(item===button)));update()};recent.append(button)
   }
   notice.textContent=rows?.length?"":"没有近期比赛，可通过链接导入。";update();
  }).catch(()=>{if(dialog.open&&runEpoch===importEpoch){loading=false;notice.textContent="近期比赛暂不可用，可通过链接导入。";update()}});
  if(!await confirmed)return null;
  const link=manual?url.value.trim():"https://accoding.buaa.edu.cn/contest-ng/index.html#/"+contestId;
  if(!manual&&!contestId)return null;
  let p=await withProgress("正在获取题目与样例，生成导入预览…",()=>call<Preview>("OJPreviewImportAt",link,lang.value,manual&&attach.checked,destination));const folder=el<HTMLInputElement>("ojFolderName"),field=el("ojFolderField"),error=el("ojFolderError");let replanBusy=false;field.hidden=!p.plan.canRename;folder.value=p.plan.folderName??"";folder.disabled=false;error.textContent="";el("ojPreviewTitle").textContent=p.plan.title;el("ojPreviewDir").textContent=p.plan.directory;
  function drawPlan(){const list=el("ojPreviewRows");list.replaceChildren();let count=0;for(const row of p.plan.rows){const item=document.createElement("li");const action=row.action==="create"?"新建":row.action==="attach"?"关联":row.action==="repair"?"补充":"跳过";if(row.action!=="skip")count++;item.textContent=`${action} ${row.name} · ${row.title} · ${row.samples} 组样例${row.document?" · "+row.document:""}${row.warnings.length?" · "+row.warnings.join("；"):""}`;list.append(item)}btn("ojPreviewConfirm").disabled=count===0||replanBusy||(!field.hidden&&folder.value!==p.plan.folderName)}drawPlan();
  folder.oninput=()=>{error.textContent="";drawPlan()};
  folder.onchange=async()=>{if(replanBusy||folder.value===p.plan.folderName)return;replanBusy=true;folder.disabled=true;btn("ojPreviewConfirm").disabled=true;try{const next=await call<Preview>("OJReplanImport",p.token,folder.value);if(preview.open&&runEpoch===importEpoch){p=next;folder.value=p.plan.folderName??"";field.hidden=!p.plan.canRename;error.textContent=""}}catch(e){if(preview.open&&runEpoch===importEpoch)error.textContent=String(e)}finally{replanBusy=false;if(runEpoch===importEpoch){folder.disabled=false;drawPlan()}}};
  if(!await modal(preview)||replanBusy||(!field.hidden&&folder.value!==p.plan.folderName))return null;return await withProgress("正在导入题目，请稍候…",()=>call<Apply<T>>("OJApplyImport",p.token))
 }
 return {refresh,run,clearBinding};
}

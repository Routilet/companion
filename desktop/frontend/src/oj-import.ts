type Call = <T>(method:string,...args:unknown[])=>Promise<T>;
type Row={name:string;title:string;action:"create"|"attach"|"skip";samples:number;warnings:string[]};
type Preview={token:string;plan:{title:string;directory:string;rows:Row[]}};
type Binding={title:string;label:string;sourceUrl:string;target:{kind:string}};
type Apply<T>={nav:T;completed:string[];skipped:string[];failed:string[]};
const el=<T extends HTMLElement=HTMLElement>(id:string)=>document.getElementById(id) as T;
const btn=(id:string)=>el<HTMLButtonElement>(id);
function modal(dialog:HTMLDialogElement):Promise<boolean>{return new Promise(resolve=>{dialog.returnValue="";dialog.addEventListener("close",()=>resolve(dialog.returnValue==="confirm"),{once:true});dialog.showModal()})}
export function installOJImport<T>(call:Call){
 const dialog=el<HTMLDialogElement>("ojImportDialog"),preview=el<HTMLDialogElement>("ojImportPreview");
 async function refresh(path:string){
  const binding=el("ojBinding");binding.hidden=true;el("ojBindingError").hidden=true;
  try{const a=await call<{available:boolean}>("OJAccountStatus");btn("ojImport").hidden=!a.available}catch{btn("ojImport").hidden=true}
  if(!path)return;try{const b=await call<Binding|null>("OJBinding");if(!b)return;el("ojBindingTitle").textContent=`Accoding · ${b.label} · ${b.title}`;btn("ojOpenOriginal").textContent=b.target.kind==="contest"?"打开原比赛 ↗":"打开原题 ↗";binding.hidden=false}catch{el("ojBindingError").textContent="题目关联无法读取，原文件保留。";el("ojBindingError").hidden=false}
 }
 async function run(path:string):Promise<Apply<T>|null>{
  const a=await call<{available:boolean;state:string}>("OJAccountStatus");if(!a.available)throw Error("Accoding Adapter 不可用");if(a.state!=="signed_in"&&a.state!=="unverified")throw Error("请先在设置中登录 Accoding");
  const url=el<HTMLInputElement>("ojImportURL"),lang=el<HTMLSelectElement>("ojImportLang"),attach=el<HTMLInputElement>("ojImportAttach");url.value="";attach.checked=false;attach.disabled=!path;lang.value=path&&!/\.c$/i.test(path)?".cpp":".c";
  if(!await modal(dialog))return null;const p=await call<Preview>("OJPreviewImport",url.value,lang.value,attach.checked);el("ojPreviewTitle").textContent=p.plan.title;el("ojPreviewDir").textContent=p.plan.directory;
  const list=el("ojPreviewRows");list.replaceChildren();let count=0;for(const row of p.plan.rows){const item=document.createElement("li");const action=row.action==="create"?"新建":row.action==="attach"?"关联":"跳过";if(row.action!=="skip")count++;item.textContent=`${action} ${row.name} · ${row.title} · ${row.samples} 组样例${row.warnings.length?" · "+row.warnings.join("；"):""}`;list.append(item)}btn("ojPreviewConfirm").disabled=count===0;if(!await modal(preview))return null;return await call<Apply<T>>("OJApplyImport",p.token)
 }
 return {refresh,run};
}

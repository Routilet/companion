type Call=<T>(method:string,...args:unknown[])=>Promise<T>;
type Row={path:string;version:string;kind:"template"|"modified"|"attach"|"current"|"skip";reason?:string};
type Preview={token:string;plan:{version:string;rows:Row[]}};
const el=<T extends HTMLElement=HTMLElement>(id:string)=>document.getElementById(id) as T;
export function installKitSwitch(call:Call){
 const dialog=el<HTMLDialogElement>("kitDialog"),rows=el("kitRows"),apply=el<HTMLButtonElement>("kitApply"),cancel=el<HTMLButtonElement>("kitCancel"),notice=el("kitSwitchNotice");
 async function run(path:string,group:string[]=[]){
  rows.replaceChildren();notice.textContent="";el("kitTargetVersion").textContent="正在读取已安装资源…";apply.disabled=true;cancel.disabled=false;dialog.showModal();
  try{
   const p=await call<Preview>("PreviewKitSwitch",path,group);if(!dialog.open)return null;el("kitTargetVersion").textContent="已安装版本 · "+p.plan.version;
   const selected=new Set<string>(),single=p.plan.rows.length===1;
   function paint(){apply.disabled=!selected.size;apply.textContent="切换 "+selected.size+" 个文件"}
   for(const r of p.plan.rows){const line=document.createElement("label");line.className="kit-row";const box=document.createElement("input");box.type="checkbox";box.disabled=r.kind==="current"||r.kind==="skip";box.checked=!box.disabled&&(single||r.kind==="template");if(box.checked)selected.add(r.path);box.onchange=()=>{if(box.checked)selected.add(r.path);else selected.delete(r.path);paint()};const text=document.createElement("span");const kind={template:"与初始模板一致",modified:"已修改或无法确认初始状态",attach:"未接入",current:"已使用",skip:"不可切换"}[r.kind];text.textContent=r.path+" · "+(r.version||"—")+" · "+kind+(r.reason?"："+r.reason:"");line.append(box,text);rows.append(line)}
   paint();const confirmed=await new Promise<boolean>(resolve=>{dialog.addEventListener("close",()=>resolve(dialog.returnValue==="apply"),{once:true});dialog.returnValue="";apply.onclick=()=>dialog.close("apply");cancel.onclick=()=>dialog.close("cancel")});
   if(!confirmed||!selected.size)return null;
   return await call<{changed:string[];failed:string[]}>("ApplyKitSwitch",p.token,[...selected])
  }catch(e){if(dialog.open){notice.textContent=String(e);cancel.onclick=()=>dialog.close("cancel");return null}throw e}
 }
 cancel.onclick=()=>dialog.close("cancel");return {run};
}

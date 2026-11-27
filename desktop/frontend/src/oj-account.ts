type View={state:"signed_in"|"signed_out"|"unverified";id?:string;nickname?:string;available:boolean;canRemember:boolean;saved:boolean;warning?:string;unavailableReason?:string};
type Call=<T>(method:string,...args:unknown[])=>Promise<T>;
export function installOJAccount(call:Call,dialog:HTMLDialogElement,onAvailability:()=>void){
 const el=<T extends HTMLElement=HTMLElement>(id:string)=>document.getElementById(id) as T;
 const btn=(id:string)=>el<HTMLButtonElement>(id),field=el<HTMLFieldSetElement>("ojAccountFields"),user=el<HTMLInputElement>("ojUsername"),pass=el<HTMLInputElement>("ojPassword"),remember=el<HTMLInputElement>("ojRemember"),note=el("ojAccountNotice"),state=el("ojAccountState"),inline=el("ojAccountInline");
 let view:View|undefined,working:Promise<void>|undefined,attempted=false,notice="";
 function draw(){
  const pending=!!working,available=!!view?.available,signed=view?.state==="signed_in"||view?.state==="unverified";
  const text=pending?"正在连接…":!available?(view?.unavailableReason||"Adapter 不可用"):signed?(view?.nickname||"Accoding")+" · "+(view?.state==="unverified"?"未核验":"已登录"):"登录 Accoding";
  btn("ojAccountEntry").textContent=text;state.textContent=text;inline.textContent=notice;inline.hidden=!notice;note.textContent=notice;
  field.disabled=pending||!available;remember.disabled=!view?.canRemember||pending;
  btn("ojLogin").disabled=pending||!available;btn("ojCheckAgain").disabled=pending;
  btn("ojRestore").hidden=!view?.canRemember||!!signed;btn("ojRestore").disabled=pending||!available;
  btn("ojRefresh").hidden=!signed;btn("ojRefresh").disabled=pending||!available;
  btn("ojLogout").hidden=!signed&&!view?.canRemember;btn("ojLogout").disabled=pending;
  el("adapterConfigStatus").textContent=available?"可用":view?.unavailableReason||"不可用";
 }
 function set(v:View){view=v;draw()}
 async function status(probe=false){try{view=await call<View>(probe?"OJCheckAdapter":"OJAccountStatus")}catch(e){view=undefined;notice=String(e)}draw();return view}
 async function perform(method:string,args:unknown[]=[]){
  if(working){await working;return}
  const task=(async()=>{notice="";try{view=await call<View>(method,...args);notice=view.warning||""}catch(e){notice=String(e);try{view=await call<View>("OJAccountStatus")}catch{/* preserve error */}}})();
  working=task;draw();try{await task}finally{working=undefined;pass.value="";draw();onAvailability()}
 }
 let entering:Promise<void>|undefined;async function enter(){if(entering)return entering;if(attempted)return;attempted=true;const task=(async()=>{const v=await status();if(v?.available&&v.state==="signed_out"&&v.canRemember)await perform("OJRestore")})();entering=task;try{await task}finally{entering=undefined;onAvailability()}}
 function open(){if(!dialog.open)dialog.showModal();pass.value="";if(!working)void status()}
 async function requireLogin(){if(entering)await entering;if(working)await working;const v=await status();if(v?.available&&(v.state==="signed_in"||v.state==="unverified"))return true;open();await new Promise<void>(r=>dialog.addEventListener("close",()=>r(),{once:true}));if(working)await working;const after=await status();return !!after?.available&&(after.state==="signed_in"||after.state==="unverified")}
 btn("ojAccountEntry").onclick=open;btn("ojAccountClose").onclick=()=>dialog.close();
 btn("ojCheckAgain").onclick=()=>void status(true).then(onAvailability);
 btn("componentCheck").onclick=()=>void status(true).then(onAvailability);
 btn("ojLogin").onclick=()=>{const u=user.value.trim(),p=pass.value;if(!u||!p){notice="请输入账号和密码";draw();return}attempted=true;pass.value="";void perform("OJLogin",[u,p,remember.checked]).then(()=>{if(view?.state==="signed_in"&&dialog.open&&!notice)dialog.close()})};
 btn("ojRestore").onclick=()=>{attempted=true;void perform("OJRestore").then(()=>{if(view?.state==="signed_in"&&dialog.open&&!notice)dialog.close()})};
 btn("ojRefresh").onclick=()=>void perform("OJRefresh");
 btn("ojLogout").onclick=()=>{attempted=true;void perform("OJLogout")};
 dialog.addEventListener("close",()=>{pass.value="";user.value="";remember.checked=false});
 return {enter,requireLogin,set,status,open,connecting:()=>!!working||!!entering};
}

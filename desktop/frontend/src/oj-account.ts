// Settings-only account management. No automatic launch, keychain access or login.
type View = {state:"signed_in"|"signed_out"|"unverified";id?:string;nickname?:string;available:boolean;canRemember:boolean;saved:boolean;warning?:string;unavailableReason?:string};
type Call = <T>(method:string,...args:unknown[])=>Promise<T>;
export function installOJAccount(call:Call, dialog:HTMLDialogElement){
 const el=<T extends HTMLElement>(id:string)=>document.getElementById(id) as T;
 const field=el<HTMLFieldSetElement>("ojAccountFields"), user=el<HTMLInputElement>("ojUsername"), pass=el<HTMLInputElement>("ojPassword"), remember=el<HTMLInputElement>("ojRemember"), note=el("ojAccountNotice"), state=el("ojAccountState");
 const login=el<HTMLButtonElement>("ojLogin"),restore=el<HTMLButtonElement>("ojRestore"),refresh=el<HTMLButtonElement>("ojRefresh"),logout=el<HTMLButtonElement>("ojLogout");
 let view:View|undefined,working=false;
 function clearPassword(){pass.value="";remember.checked=false;}
 function draw(){
  field.disabled=working||!view?.available;
  remember.disabled=!view?.canRemember||working;
  restore.hidden=!view?.canRemember;restore.disabled=working||!view?.available;
  refresh.hidden=!view||view.state==="signed_out";refresh.disabled=working||!view?.available;
  logout.hidden=!view?.canRemember&&(!view||view.state==="signed_out");logout.disabled=working;
  login.disabled=working||!view?.available;
  state.textContent=!view?"正在读取账号状态…":!view.available?(view.unavailableReason||"请通过 Routilet 安装并启用 Adapter；本地功能不受影响。"):view.state==="signed_out"?"未登录":`${view.nickname||"账号"} · ${view.id||""} · ${view.state==="unverified"?"未能核验":"本次会话已登录"}`;
  if(view?.saved&&view.state!=="signed_out")state.textContent+=" · 已保存凭据";
 }
 async function update(check=false){try{view=await call<View>(check?"OJCheckAdapter":"OJAccountStatus");}catch{view=undefined;note.textContent="无法读取账号状态";}draw();}
 async function perform(method:string,args:unknown[]=[]){
  if(working)return;working=true;note.textContent="正在处理…";draw();
  try{view=await call<View>(method,...args);note.textContent=view.warning|| (method==="OJLogout"?"已注销并清理已保存凭据。":"操作完成。")}
  catch(e){note.textContent=String(e);await update();}
  finally{working=false;clearPassword();draw();}
 }
 login.onclick=()=>{const name=user.value.trim(),secret=pass.value;if(!name||!secret){note.textContent="请输入账号和密码。";return;}const save=remember.checked;clearPassword();void perform("OJLogin",[name,secret,save]);};
 restore.onclick=()=>void perform("OJRestore");
 refresh.onclick=()=>void perform("OJRefresh");
 logout.onclick=()=>void perform("OJLogout");
 dialog.addEventListener("close",()=>{clearPassword();user.value="";});
 // Closing settings never cancels or retries a request. Disable reopening actions
 // until the active operation completes; the rest of Companion is independent.
 return {open(){clearPassword();if(!working){note.textContent="";void update(true);}else draw();}};
}

// Embedded WebViews need not implement window.prompt/confirm. Use the
// application's own DOM dialog and resolve only after explicit user action.
export function installDialogs(){
 const dialog=document.getElementById("actionDialog") as HTMLDialogElement;
 const form=document.getElementById("actionForm") as HTMLFormElement;
 const title=document.getElementById("actionTitle")!;
 const message=document.getElementById("actionMessage")!;
 const field=document.getElementById("actionField")!;
 const input=document.getElementById("actionName") as HTMLInputElement;
 const cancel=document.getElementById("actionCancel") as HTMLButtonElement;
 const accept=document.getElementById("actionAccept") as HTMLButtonElement;
 const location=document.getElementById("actionLocation")!,directory=document.getElementById("actionDirectory")!,browse=document.getElementById("actionBrowse") as HTMLButtonElement;
 let pending=false;
 function ask(heading:string,detail:string,action:string,initial?:string,place?:{value:string;choose:()=>Promise<string>}):Promise<string|null>{
  if(pending||dialog.open)return Promise.resolve(null);
  const named=initial!==undefined;
  title.textContent=heading;message.textContent=detail;accept.textContent=action;
  field.hidden=!named;input.disabled=!named;input.required=named;input.value=initial??"";
  location.hidden=!place;directory.textContent=place?.value||"请选择文件夹";directory.title=place?.value??"";accept.disabled=!!place&&!place.value;
  dialog.returnValue="";pending=true;
  return new Promise((resolve,reject)=>{
   let submitted:string|null=null;
   function cleanup(){
    pending=false;browse.onclick=null;accept.disabled=false;form.removeEventListener("submit",submit);cancel.removeEventListener("click",dismiss);
    dialog.removeEventListener("close",closed);dialog.removeEventListener("cancel",escape);
   }
   function submit(e:Event){
    e.preventDefault();
    const value=named?input.value:"yes";
    if(place&&!place.value)return;
    if(named&&!value.trim()){input.focus();return}
    submitted=value;dialog.close("accept");
   }
   function dismiss(){submitted=null;dialog.close("cancel")}
   function escape(e:Event){e.preventDefault();dismiss()}
   function closed(){const value=dialog.returnValue==="accept"?submitted:null;cleanup();resolve(value)}
   if(place)browse.onclick=async()=>{browse.disabled=true;accept.disabled=true;try{const chosen=await place.choose();if(chosen&&dialog.open){place.value=chosen;directory.textContent=chosen;directory.title=chosen}}catch(e){message.textContent=String(e)}finally{browse.disabled=false;accept.disabled=!place.value}};
   form.addEventListener("submit",submit);cancel.addEventListener("click",dismiss);
   dialog.addEventListener("close",closed);dialog.addEventListener("cancel",escape);
   try{dialog.showModal();if(named){input.focus();input.select()}else cancel.focus()}
   catch(e){cleanup();reject(e)}
  });
 }
 return {isOpen:()=>pending||dialog.open,
  name:(heading:string,detail:string,initial:string,action:string)=>ask(heading,detail,action,initial),
  create:async(initialDir:string,choose:()=>Promise<string>)=>{const place={value:initialDir,choose};const name=await ask("新建文件","C 默认引入 Exam Kit，C++ 为普通模板","新建","A.c",place);return name?{name,dir:place.value}:null},
  confirm:async(detail:string,action="确认")=>(await ask("确认操作",detail,action))!==null};
}

type Call=<T>(method:string,...args:unknown[])=>Promise<T>;
import {renderJudge,type Judge} from "./judge-detail.js";
type View={status:{accepted:boolean;latest?:Judge};checked:string};
const el=(id:string)=>document.getElementById(id)!;
export function installOJStatus(call:Call,paths:()=>string[],selected:()=>string,draw:()=>void,notify:(s:string)=>void,canPoll:()=>boolean){
 let cache:Record<string,View>={},epoch=0,pollEpoch=0;
 const detail=el("ojDetailDialog") as HTMLDialogElement;el("ojDetailClose").onclick=()=>detail.close();
 el("ojDetailBtn").onclick=()=>{const j=cache[selected()]?.status.latest;if(j){renderJudge(el("ojDetailBody"),j);detail.showModal()}};
 function paint(){const v=cache[selected()];const area=el("ojJudgeInfo");area.hidden=!selected()||el("ojBinding").hidden||el("ojImport").hidden;el("ojJudgeText").textContent=v?`${v.status.accepted?"曾 AC":"当前记录未发现 AC"} · 最近可见提交 ${v.status.latest?"#"+v.status.latest.id+" · "+v.status.latest.result:"无"} · ${new Date(v.checked).toLocaleTimeString()}`:"尚未刷新在线评测";el("ojDetailBtn").hidden=area.hidden||!v?.status.latest;if(detail.open&&v?.status.latest)renderJudge(el("ojDetailBody"),v.status.latest);draw()}
 async function refreshLocal(){if(el("ojImport").hidden){clear();return}const n=++epoch;try{const a=await call<{available:boolean}>("OJAccountStatus");if(n!==epoch)return;if(!a.available){clear();return}const p=Array.from(new Set([selected(),...paths()].filter(Boolean))).slice(0,512);const v=await call<Record<string,View>>("OJCachedJudges",p);if(n===epoch){cache=v??{};paint()}}catch{if(n===epoch){cache={};paint()}}}
 function clear(){epoch++;pollEpoch++;cache={};paint()}
 function mark(path:string){if(el("ojImport").hidden)return "";const v=cache[path];return v?(v.status.accepted?"AC":v.status.latest?.result??""):""}
 async function refresh(list:string[]){if(el("ojImport").hidden)return;pollEpoch++;try{const n=await call<number>("OJRefreshJudges",list);notify(`已刷新 ${n} 道关联题目`)}catch(e){notify(String(e))}finally{await refreshLocal()}}
 function poll(id:string,result:string){const n=++pollEpoch;const until=Date.now()+60000;void refreshLocal();if(result!=="WT"&&result!=="JG"){void final();return}
  async function final(){try{const v=await call<Judge>("OJPollReceipt",id);if(n!==pollEpoch)return;await refreshLocal();notify(`提交 #${id} · ${v.result}`)}catch{/* manual refresh remains available */}}
  async function step(){if(el("ojImport").hidden){return}if(n!==pollEpoch||Date.now()>=until){if(n===pollEpoch)notify("自动查询已停止，可手动刷新评测结果");return}if(!canPoll()){setTimeout(()=>void step(),2000);return}
   try{const v=await call<Judge>("OJPollReceipt",id);if(n!==pollEpoch)return;await refreshLocal();notify(`提交 #${id} · ${v.result}`);if(v.result==="WT"||v.result==="JG")setTimeout(()=>void step(),2000)}catch(e){if(n===pollEpoch)notify("自动查询停止："+String(e))}}
  setTimeout(()=>void step(),2000)
 }
 return {refreshLocal,clear,mark,refresh,poll};
}

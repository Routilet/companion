type Call=<T>(method:string,...args:unknown[])=>Promise<T>;
import {renderJudge,parseDetail,type Judge} from "./judge-detail.js";
type View={status:{accepted:boolean;latest?:Judge};checked:string};
const el=(id:string)=>document.getElementById(id)!;
export function installOJStatus(call:Call,paths:()=>string[],selected:()=>string,draw:()=>void,notify:(s:string)=>void,canPoll:()=>boolean){
 let cache:Record<string,View>={},epoch=0,pollEpoch=0;
 const detail=el("ojDetailDialog") as HTMLDialogElement;el("ojDetailClose").onclick=()=>detail.close();
 el("ojDetailBtn").onclick=()=>{const j=cache[selected()]?.status.latest;if(j)renderJudge(el("ojDetailBody"),j);else el("ojDetailBody").textContent="尚无可见提交结果";el("ojDetailMeta").textContent=cache[selected()]?"查询时间："+new Date(cache[selected()].checked).toLocaleString():"尚未查询";detail.showModal()};
 function paint(){const v=cache[selected()];const area=el("ojJudgeInfo");area.hidden=!selected()||el("ojBinding").hidden||el("ojImport").hidden;const latest=v?.status.latest,rows=latest?parseDetail(latest):null;el("ojJudgeText").textContent=v?[v.status.accepted?"曾通过":"",latest?.result??"无提交",rows?rows.filter(r=>r.code==="AC").length+"/"+rows.length:"",latest?.score??""].filter(Boolean).join(" · "):"尚未查询评测";el("ojDetailBtn").hidden=area.hidden;el("ojDetailBtn").title="在线评测详情；曾通过不代表当前代码正确";if(detail.open){if(v?.status.latest)renderJudge(el("ojDetailBody"),v.status.latest);else el("ojDetailBody").textContent="尚无可见提交结果";el("ojDetailMeta").textContent=v?"查询时间："+new Date(v.checked).toLocaleString():"尚未查询"}draw()}
 async function refreshLocal(){if(el("ojImport").hidden){clear();return}const n=++epoch;try{const a=await call<{available:boolean}>("OJAccountStatus");if(n!==epoch)return;if(!a.available){clear();return}const p=Array.from(new Set([selected(),...paths()].filter(Boolean))).slice(0,512);const v=await call<Record<string,View>>("OJCachedJudges",p);if(n===epoch){cache=v??{};paint()}}catch{if(n===epoch){cache={};paint()}}}
 function clear(){epoch++;pollEpoch++;cache={};if(detail.open)detail.close();el("ojDetailBody").replaceChildren();paint()}
 function mark(path:string){if(el("ojImport").hidden)return "";const v=cache[path];return v?(v.status.accepted?"AC":v.status.latest?.result??"未提交"):""}
 async function refresh(list:string[]){if(el("ojImport").hidden)return;pollEpoch++;try{const n=await call<number>("OJRefreshJudges",list);notify(`已刷新 ${n} 道题目的评测结果`)}catch(e){notify(String(e))}finally{await refreshLocal()}}
 function poll(id:string,result:string,onResult?:(j:Judge)=>void){if(el("ojImport").hidden)return;const n=++pollEpoch;const until=Date.now()+60000;void refreshLocal();if(result!=="WT"&&result!=="JG"){void final();return}
  async function final(){try{const v=await call<Judge>("OJPollReceipt",id);if(n!==pollEpoch)return;await refreshLocal();if(n!==pollEpoch)return;notify(`提交 #${id} · ${v.result}`);onResult?.(v)}catch{/* manual refresh remains available */}}
  async function step(){if(el("ojImport").hidden){return}if(n!==pollEpoch||Date.now()>=until){if(n===pollEpoch)notify("暂未获得最终评测结果，可手动刷新");return}if(!canPoll()){setTimeout(()=>void step(),2000);return}
   try{const v=await call<Judge>("OJPollReceipt",id);if(n!==pollEpoch)return;await refreshLocal();notify(`提交 #${id} · ${v.result}`);if(v.result==="WT"||v.result==="JG")setTimeout(()=>void step(),2000);else if(n===pollEpoch)onResult?.(v)}catch(e){if(n===pollEpoch)notify("自动查询停止："+String(e))}}
  setTimeout(()=>void step(),2000)
 }
 return {refreshLocal,clear,mark,refresh,poll};
}

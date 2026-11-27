export type Judge={id:string;result:string;language:string;score?:string;detail?:string;createdAt?:string};
const tips:Record<string,string>={AC:"通过全部测试点，但不代表程序一定正确。",CE:"编译失败，检查语法及编译器兼容性。",PE:"输出格式错误，检查空格、换行等。",WA:"答案错误，检查程序逻辑及输出。",TLE:"运行超时，检查算法效率或死循环。",MLE:"内存超限，检查数组大小和内存分配。",REG:"非法内存访问，检查数组越界、指针及栈溢出。",REP:"算术异常，检查除零、模零等。",OE:"其他运行错误，需结合实际反馈分析。",SIGSEGV:"非法内存访问，检查数组越界、指针及栈溢出。",SIGFPE:"算术异常，检查除零、模零等。",SIGABRT:"异常终止，检查断言失败及运行库错误。",SIGILL:"非法指令，检查未定义行为等。",WT:"等待评测。",JG:"正在评测。"};
const codes:Record<string,string>={"Accepted":"AC","Wrong Answer":"WA","Presentation Error":"PE","Time Limit Exceed":"TLE","Time Limit Exceeded":"TLE","Memory Limit Exceed":"MLE","Memory Limit Exceeded":"MLE","Compile Error":"CE","Runtime Error (SIGSEGV)":"REG","Runtime Error (SIGFPE)":"REP","Runtime Error (SIGABRT)":"OE","Runtime Error (SIGILL)":"OE"};
export function parseDetail(j:Judge){
 const lines=(j.detail??"").replace(/\r\n/g,"\n").split("\n");while(lines.at(-1)==="")lines.pop();
 if(lines.shift()!==j.id||!lines.length||lines.length>1000)return null;
 const out:{result:string;score:string;time:string;memory:string;code:string}[]=[];
 for(const line of lines){const m=line.match(/^([^|]+?)\s*\|\s*(\d+(?:\.\d+)?\s*\*\s*\(\s*\d+(?:\.\d+)?\s*\/\s*\d+(?:\.\d+)?\s*\))\s*\|\s*(\d+\s*ms)\s*\|\s*(\d+\s*KB)\s*$/);if(!m)return null;const result=m[1].trim();out.push({result,score:m[2],time:m[3],memory:m[4],code:codes[result]??"?"})}return out
}
function node<K extends keyof HTMLElementTagNameMap>(tag:K,text=""){const e=document.createElement(tag);e.textContent=text;return e}
export function renderJudge(host:HTMLElement,j:Judge){
 host.replaceChildren();const rows=parseDetail(j);const title=node("h3",j.result+" · #"+j.id);host.append(title);
 const meta=node("p",[j.score!==undefined&&j.score!==""?"得分 "+j.score.replace(/^"|"$/g,""):null,j.language,j.createdAt?new Date(j.createdAt).toLocaleString():null].filter(Boolean).join(" · "));host.append(meta);
 const signals=new Set<string>();for(const v of rows??[]){const m=v.result.match(/\b(SIGSEGV|SIGFPE|SIGABRT|SIGILL)\b/);if(m)signals.add(m[1])}
 if(signals.size){for(const s of signals)host.append(node("p",s+"："+tips[s]))}else if(tips[j.result])host.append(node("p",tips[j.result]));
 if(rows){
  const passed=rows.filter(x=>x.code==="AC").length;host.append(node("p",passed+" / "+rows.length+" 个评测点通过"));
  const strip=node("div");strip.className="judge-strip";for(const [i,r] of rows.entries()){const b=node("span",String(i+1)+" "+r.code);b.className=r.code==="AC"?"judge-pass":"judge-other";b.title=r.result;strip.append(b)}host.append(strip);
  const table=node("table"),header=node("tr");for(const h of ["评测点","结果","得分计算","耗时","内存"])header.append(node("th",h));table.append(header);
  rows.forEach((r,i)=>{const tr=node("tr");for(const v of [String(i+1),r.result,r.score,r.time,r.memory])tr.append(node("td",v));tr.className=r.code==="AC"?"passed":"";table.append(tr)});host.append(table)
 }else{host.append(node("p",j.detail?"评测详情格式未识别，保留原文。":"暂无逐点评测详情。"));if(j.detail)host.append(node("pre",j.detail))}
}

// Offline-only Markdown rendering. No raw HTML, clickable URL, images, or
// external resource loads. KaTeX is bundled at build time with trust disabled.
declare global {interface Window {markdownit:(options:Record<string,unknown>)=>any;katex:{renderToString:(s:string,options:Record<string,unknown>)=>string}}}
type Call=<T>(method:string,...args:unknown[])=>Promise<T>;
type DocumentData={name:string;content:string};
const el=<T extends HTMLElement=HTMLElement>(id:string)=>document.getElementById(id) as T;
// Some imported inline formulas contain a doubled escape before common
// relation commands. Do not rewrite display math or multi-line environments.
export function normalizeInlineMath(value:string,display:boolean):string{
 if(display||/\\(?:begin|end)\s*\{|[\r\n]/.test(value))return value;
 return value.replace(/\\\\(leqslant|geqslant|leq|geq|le|ge|sim|neq|ne|lt|gt|approx|times|cdot|pm|in|notin)(?=\s|[0-9{}_^]|$)/g,"\\$1");
}
export function installDocument(call:Call){
 const button=el<HTMLButtonElement>("documentBtn"),dialog=el<HTMLDialogElement>("documentDialog"),body=el("documentBody");
 let data:DocumentData|null=null,epoch=0;
 function render(source:string){
  const md=window.markdownit({html:false,linkify:false,breaks:false,maxNesting:32});
  md.renderer.rules.image=()=>'<span class="image-placeholder">[此处有一张图片，未加载]</span>';
  md.renderer.rules.link_open=()=>'<span class="inert-link">';
  md.renderer.rules.link_close=()=>'</span>';
  // Recognize math before Markdown escape/emphasis processing. Code spans and
  // fenced/indented blocks are handled by Markdown itself, never by KaTeX.
  md.inline.ruler.before("escape","local_math",(state:any,silent:boolean)=>{
   const start=state.pos,s=state.src as string;
   let open="",close="";
   if(s.startsWith("$$",start)){open="$$";close="$$"}else if(s[start]==="$"){open="$";close="$"}else if(s.startsWith("\\(",start)){open="\\(";close="\\)"}else if(s.startsWith("\\[",start)){open="\\[";close="\\]"}else{return false}
   let end=start+open.length;while((end=s.indexOf(close,end))>=0){if(s[end-1]!=="\\"||close.startsWith("\\"))break;end+=close.length}
   if(end<0||end>=state.posMax||end===start+open.length||open==="$"&&s.slice(start+1,end).includes("\n"))return false;
   if(!silent){const token=state.push("local_math","",0);token.content=s.slice(start+open.length,end);token.meta={display:open==="$$"||open==="\\["}}
   state.pos=end+close.length;return true
  });
  md.renderer.rules.local_math=(tokens:any[],i:number)=>{
   try{return window.katex.renderToString(normalizeInlineMath(tokens[i].content,tokens[i].meta.display),{displayMode:tokens[i].meta.display,throwOnError:true,trust:false,maxExpand:300,maxSize:10,strict:"ignore"})}catch{return md.utils.escapeHtml(tokens[i].content)}
  };
  // Rendering is confined to a single inert article. Markdown-it escapes all
  // source HTML; generated links and images are deliberately replaced above.
  return md.render(source)
 }
 async function refresh(path:string){
  const n=++epoch;data=null;button.hidden=true;el("title").classList.remove("document-link");if(!path)return;
  try{const d=await call<DocumentData|null>("ReadLocalDocument");if(n===epoch){data=d;button.hidden=!d;el("title").classList.toggle("document-link",!!d);el("title").title=d?"阅读题面 / 文档":""}}catch{/* optional local document, ordinary testing unaffected */}
 }
 button.onclick=async()=>{try{const d=await call<DocumentData|null>("ReadLocalDocument");if(!d)return;data=d;el("documentTitle").textContent=el("title").textContent||data.name;body.replaceChildren();body.scrollTop=0;try{body.innerHTML=render(data.content)}catch{const pre=document.createElement("pre");pre.textContent=data.content;body.append(pre)}dialog.showModal()}catch(e){body.textContent=String(e);dialog.showModal()}};
 el("documentClose").onclick=()=>dialog.close();
 return {refresh,isOpen:()=>dialog.open}
}

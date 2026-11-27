type Variant={key:string;label:string;symbols:string[];signatures:string[];advancedSignatures?:string[];note:string;example:string};
type Entry={id:string;title:string;description:string;example:string;signatures:string[];commonSignatures?:string[];advancedSignatures?:string[];related?:string[];variants?:Variant[]};
type Section={title:string;children?:Section[];entries?:string[]};
type Reference={version:string;notes:string;categories:{title:string;entries:Entry[]}[];sections:Section[]};
type Manual={reference:Reference;origin:string;sourcePath?:string;fingerprint:string};
type Hit={id:string;variant?:string;category:string};
type Call=<T>(method:string,...args:unknown[])=>Promise<T>;
const el=<T extends HTMLElement=HTMLElement>(id:string)=>document.getElementById(id) as T;
const node=(tag:string,text="",cls="")=>{const n=document.createElement(tag);n.textContent=text;if(cls)n.className=cls;return n};
const button=(text:string,action:()=>void)=>{const b=document.createElement("button");b.textContent=text;b.type="button";b.onclick=action;return b};
function inline(parent:HTMLElement,text:string){
 // Only the inline syntax used in resource prose. Never interpret HTML or URLs.
 const pattern=/(\x60[^\x60\n]+\x60|\*\*[^*\n]+\*\*)/g;
 let last=0;
 for(const m of text.matchAll(pattern)){
  parent.append(document.createTextNode(text.slice(last,m.index)));
  parent.append(m[0].startsWith("\x60")?node("code",m[0].slice(1,-1)):node("strong",m[0].slice(2,-2)));
  last=(m.index??0)+m[0].length;
 }
 parent.append(document.createTextNode(text.slice(last)));
}
function prose(parent:HTMLElement,text:string){
 // Small safe renderer for headings, paragraphs, bullets and fenced code.
 // All content comes from text nodes. Unknown Markdown remains visible.
 let fence=false,code:string[]=[];
 for(const line of text.split(/\r?\n/)){
  if(line.startsWith("\x60\x60\x60")){
   if(fence){parent.append(node("pre",code.join("\n")));code=[]}fence=!fence;continue
  }
  if(fence){code.push(line);continue}
  if(!line.trim())continue;
  const heading=/^#{1,6}\s+(.+)$/.exec(line);
  const p=node(heading?"h3":"p");inline(p,heading?heading[1]:line);parent.append(p)
 }
 if(code.length)parent.append(node("pre",code.join("\n")));
}
export function installManual(call:Call,canOpen:()=>boolean){
 const dialog=el<HTMLDialogElement>("manualDialog"),search=el<HTMLInputElement>("manualSearch"),nav=el("manualNav"),detail=el("manualDetail"),status=el("manualStatus"),version=el("manualVersion");
 let model:Manual|undefined,entries=new Map<string,Entry>(),generation=0,queryEpoch=0,current="",currentKey="",trail:{id:string;key:string}[]=[];
 function setStatus(s:string){status.textContent=s}
 function clear(){nav.replaceChildren();detail.replaceChildren();current="";currentKey="";trail=[]}
 function highlight(){for(const b of Array.from(nav.querySelectorAll<HTMLButtonElement>("[data-entry]")))b.setAttribute("aria-current",String(b.dataset.entry===current))}
 function notes(){
  current="";currentKey="";highlight();detail.replaceChildren(node("h2","使用约定"));
  if(model)prose(detail,model.reference.notes||"此版本没有独立使用约定。");detail.scrollTop=0
 }
 function show(id:string,key="",remember=false){
  const e=entries.get(id);if(!e||!model)return;
  if(remember&&current)trail.push({id:current,key:currentKey});
  const v=e.variants?.find(v=>v.key===key)??e.variants?.[0];
  current=id;currentKey=v?.key??"";highlight();
  detail.replaceChildren();const head=node("div","","manualHead");
  head.append(node("h2",e.title));if(trail.length)head.append(button("返回",()=>{const prev=trail.pop()!;show(prev.id,prev.key)}));detail.append(head);
  if(e.variants?.length){
   const variants=node("div","","manualVariants");
   for(const item of e.variants){const b=button(item.label,()=>show(id,item.key));b.setAttribute("aria-pressed",String(item.key===v?.key));variants.append(b)}
   detail.append(variants)
  }
  const signatures=v?[...(e.commonSignatures??[]),...v.signatures]:e.signatures;
  const advanced=[...new Set([...(e.advancedSignatures??[]),...(v?.advancedSignatures??[])])];
  const visible=signatures.filter(s=>!advanced.includes(s));
  if(visible.length)detail.append(node("h3","函数接口"),node("pre",visible.join("\n")));
  detail.append(node("h3","使用说明"));prose(detail,e.description);if(v?.note)prose(detail,v.note);
  const example=v?.example??e.example;
  if(example){
   const row=node("div","","manualHead");row.append(node("h3","调用示例"));
   const fp=model.fingerprint,selectedKey=currentKey;
   const copy=button("复制示例",()=>{copy.disabled=true;void call("CopyManualExample",id,selectedKey,fp).then(()=>setStatus("已复制完整示例")).catch(err=>setStatus(String(err))).finally(()=>{copy.disabled=false})});
   row.append(copy);detail.append(row,node("pre",example))
  }
  if(advanced.length){const more=document.createElement("details");more.append(node("summary","扩展接口 · 显式容量"),node("pre",advanced.join("\n")));detail.append(more)}
  if(e.related?.length){detail.append(node("h3","相关操作"));const related=node("div","","manualVariants");for(const id of e.related){const target=entries.get(id);if(target)related.append(button(target.title,()=>show(id,"",true)))}detail.append(related)}
  detail.scrollTop=0
 }
 function entryButton(id:string,key="",category=""){
  const e=entries.get(id);if(!e)return node("span",id);
  const b=button(e.title,()=>{trail=[];show(id,key)});b.dataset.entry=id;b.title=category||e.id;if(category)b.append(node("small",category));return b
 }
 function tree(){
  if(!model)return;nav.replaceChildren(button("使用约定",notes));
  function section(n:Section):HTMLElement{
   const d=document.createElement("details");d.append(node("summary",n.title));
   const items=node("div","","manualBranch");
   for(const child of n.children??[])items.append(section(child));
   for(const id of n.entries??[])items.append(entryButton(id));d.append(items);return d
  }
  for(const s of model.reference.sections)nav.append(section(s));highlight()
 }
 async function searchNow(){
  const epoch=++queryEpoch,session=generation,q=search.value.trim();
  if(!model)return;
  if(!q){tree();return}
  try{const hits=await call<Hit[]>("SearchManual",q);if(epoch!==queryEpoch||session!==generation||!dialog.open)return;
   nav.replaceChildren(node("p",hits.length+" 项结果","muted"));
   for(const h of hits)nav.append(entryButton(h.id,h.variant,h.category));highlight()
  }catch(e){if(epoch===queryEpoch&&session===generation)setStatus(String(e))}
 }
 async function open(){
  if(dialog.open||!canOpen())return;
  const session=++generation;queryEpoch++;model=undefined;entries.clear();clear();search.value="";search.disabled=true;version.textContent="";setStatus("正在读取本地资源…");dialog.showModal();
  try{const m=await call<Manual>("OpenManual");if(session!==generation||!dialog.open)return;
   model=m;entries=new Map(m.reference.categories.flatMap(c=>c.entries.map(e=>[e.id,e] as const)));
   version.textContent="Exam Kit "+m.reference.version+(m.origin==="snapshot"?" · 当前文件资源快照":" · 已安装资源")+" · "+entries.size+" 项";
   tree();notes();search.disabled=false;setStatus("");search.focus()
  }catch(e){if(session===generation&&dialog.open){setStatus(String(e));detail.append(node("p","无法读取匹配的手册。请检查当前文件的资源快照，或在设置中准备已安装 Exam Kit。"))}}
 }
 el<HTMLButtonElement>("manualBtn").onclick=()=>void open();
 el<HTMLButtonElement>("manualClose").onclick=()=>dialog.close();
 search.oninput=()=>void searchNow();
 search.onkeydown=e=>{if(e.key==="Enter"){e.preventDefault();nav.querySelector<HTMLButtonElement>("[data-entry]")?.click()}};
 dialog.onclose=()=>{generation++;queryEpoch++;model=undefined;entries.clear()};
 return {open,isOpen:()=>dialog.open}
}

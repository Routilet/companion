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
function codeBlock(text:string){
 const pre=node("pre"),code=node("code","","manual-code");
 const re=/(\/\*[\s\S]*?\*\/|\/\/[^\n]*|"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|\b(?:void|char|short|int|long|float|double|signed|unsigned|const|struct|typedef|return|if|else|for|while|do|switch|case|break|continue|sizeof|bool|true|false|static|enum|union)\b|\b(?:0[xX][\da-fA-F]+|\d+(?:\.\d+)?)\b)/g;
 let pos=0;for(const m of text.matchAll(re)){const i=m.index??0;code.append(document.createTextNode(text.slice(pos,i)));const token=m[0];code.append(node("span",token,token.startsWith("/")?"syntax-comment":/^["']/.test(token)?"syntax-string":/^\d/.test(token)?"syntax-number":"syntax-keyword"));pos=i+token.length}code.append(document.createTextNode(text.slice(pos)));pre.append(code);return pre
}
function prose(parent:HTMLElement,text:string){
 // Small safe renderer for headings, paragraphs, bullets and fenced code.
 // All content comes from text nodes. Unknown Markdown remains visible.
 let fence=false,code:string[]=[],list:HTMLElement|undefined;
 for(const line of text.split(/\r?\n/)){
  if(line.startsWith("\x60\x60\x60")){
   if(fence){parent.append(codeBlock(code.join("\n")));code=[]}list=undefined;fence=!fence;continue
  }
  if(fence){code.push(line);continue}
  if(!line.trim()){list=undefined;continue}
  const bullet=/^\s*([-*]|\d+\.)\s+(.+)$/.exec(line);if(bullet){const tag=/\d/.test(bullet[1])?"ol":"ul";if(!list||list.tagName.toLowerCase()!==tag){list=node(tag);parent.append(list)}const li=node("li");inline(li,bullet[2]);list.append(li);continue}list=undefined;
  const heading=/^#{1,6}\s+(.+)$/.exec(line);
  const p=node(heading?"h3":"p");inline(p,heading?heading[1]:line);parent.append(p)
 }
 if(code.length)parent.append(codeBlock(code.join("\n")));
}
export function installManual(call:Call,canOpen:()=>boolean){
 const dialog=el<HTMLDialogElement>("manualDialog"),search=el<HTMLInputElement>("manualSearch"),nav=el("manualNav"),detail=el("manualDetail"),status=el("manualStatus"),version=el("manualVersion");
 let model:Manual|undefined,entries=new Map<string,Entry>(),generation=0,queryEpoch=0,current="",currentKey="",trail:{id:string;key:string}[]=[];
 const saved=new Map<string,{id:string;key:string;scroll:number;navScroll:number;expanded:string[]}>();let openSections=new Set<string>();let keyboardIndex=-1;
 function setStatus(s:string){status.textContent=s}
 function clear(){nav.replaceChildren();detail.replaceChildren();current="";currentKey="";trail=[]}
 function highlight(){for(const b of Array.from(nav.querySelectorAll<HTMLButtonElement>("[data-entry]")))b.setAttribute("aria-current",String(b.dataset.entry===current))}
 function locate(){if(search.value.trim())return;const b=Array.from(nav.querySelectorAll<HTMLButtonElement>("[data-entry]")).find(b=>b.dataset.entry===current);if(!b)return;let p=b.parentElement;while(p&&p!==nav){if(p instanceof HTMLDetailsElement){p.open=true;if(p.dataset.section)openSections.add(p.dataset.section)}p=p.parentElement}b.scrollIntoView({block:"nearest"})}
 function notes(){
  current="";currentKey="";highlight();detail.replaceChildren(node("h2","使用约定"));
  if(model)prose(detail,model.reference.notes||"此版本没有独立使用约定。");detail.scrollTop=0
 }
 function show(id:string,key="",remember=false){
  const e=entries.get(id);if(!e||!model)return;
  if(remember&&current)trail.push({id:current,key:currentKey});
  const v=e.variants?.find(v=>v.key===key)??e.variants?.[0];
  current=id;currentKey=v?.key??"";highlight();locate();
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
  function signatureList(items:string[]){
   for(const signature of items){const wrap=node("div","","signature-row");const b=button("复制",()=>{b.disabled=true;void call("CopyManualSignature",id,currentKey,model!.fingerprint,signature).then(()=>{b.textContent="已复制";setTimeout(()=>{b.textContent="复制"},1200)}).catch(err=>setStatus(String(err))).finally(()=>{b.disabled=false})});b.setAttribute("aria-label","复制函数接口");wrap.append(codeBlock(signature),b);detail.append(wrap)}
  }
  if(visible.length){detail.append(node("h3","函数接口"));signatureList(visible)}
  detail.append(node("h3","使用说明"));prose(detail,e.description);if(v?.note)prose(detail,v.note);
  const example=v?.example??e.example;
  if(example){
   const row=node("div","","manualHead");row.append(node("h3","调用示例"));
   const fp=model.fingerprint,selectedKey=currentKey;
   const copy=button("复制示例",()=>{copy.disabled=true;void call("CopyManualExample",id,selectedKey,fp).then(()=>setStatus("已复制完整示例")).catch(err=>setStatus(String(err))).finally(()=>{copy.disabled=false})});
   row.append(copy);detail.append(row,codeBlock(example))
  }
  if(advanced.length){const more=document.createElement("details");more.append(node("summary","扩展接口 · 显式容量"),codeBlock(advanced.join("\n")));detail.append(more)}
  if(e.related?.length){detail.append(node("h3","相关操作"));const related=node("div","","manualVariants");for(const id of e.related){const target=entries.get(id);if(target)related.append(button(target.title,()=>show(id,"",true)))}detail.append(related)}
  detail.scrollTop=0
 }
 function entryButton(id:string,key="",category=""){
  const e=entries.get(id);if(!e)return node("span",id);
  const b=button(e.title,()=>{trail=[];show(id,key)});b.dataset.entry=id;b.title=category||e.id;if(category){const v=e.variants?.find(v=>v.key===key);b.append(node("small",v?.symbols.join(" · ")??e.id),node("small",category))}return b
 }
 function tree(){
  if(!model)return;nav.replaceChildren(button("使用约定",notes));
  function section(n:Section,key:string):HTMLElement{
   const d=document.createElement("details");d.dataset.section=key;d.open=openSections.has(key);
   const summary=node("summary",n.title);d.append(summary);d.ontoggle=()=>{if(!d.isConnected)return;if(d.open)openSections.add(key);else openSections.delete(key)};
   const items=node("div","","manualBranch");
   for(const [i,child] of (n.children??[]).entries())items.append(section(child,key+"/"+i));
   for(const id of n.entries??[])items.append(entryButton(id));d.append(items);return d
  }
  for(const [i,s] of model.reference.sections.entries())nav.append(section(s,String(i)));highlight();locate()
 }
 async function searchNow(){
  const epoch=++queryEpoch,session=generation,q=search.value.trim();
  if(!model)return;
  keyboardIndex=-1;if(!q){tree();return}
  try{const hits=await call<Hit[]>("SearchManual",q);if(epoch!==queryEpoch||session!==generation||!dialog.open)return;
   nav.replaceChildren(node("p",hits.length+" 项结果","muted"));
   for(const h of hits)nav.append(entryButton(h.id,h.variant,h.category));highlight();
  }catch(e){if(epoch===queryEpoch&&session===generation)setStatus(String(e))}
 }
 async function open(){
  if(dialog.open||!canOpen())return;
  const session=++generation;queryEpoch++;model=undefined;entries.clear();clear();search.value="";search.disabled=true;version.textContent="";setStatus("正在读取本地资源…");dialog.showModal();
  try{const m=await call<Manual>("OpenManual");if(session!==generation||!dialog.open)return;
   model=m;entries=new Map(m.reference.categories.flatMap(c=>c.entries.map(e=>[e.id,e] as const)));
   version.textContent="Exam Kit "+m.reference.version+(m.origin==="snapshot"?" · 当前文件资源快照":" · 已安装资源")+" · "+entries.size+" 项";
   const previous=saved.get(m.fingerprint);openSections=new Set(previous?.expanded??[]);tree();if(previous?.id&&entries.has(previous.id)){show(previous.id,previous.key);detail.scrollTop=previous.scroll;nav.scrollTop=previous.navScroll}else notes();search.disabled=false;setStatus("");search.focus()
  }catch(e){if(session===generation&&dialog.open){setStatus(String(e));detail.append(node("p","无法读取匹配的手册。请检查当前文件的资源快照，或在设置中准备已安装 Exam Kit。"))}}
 }
 el<HTMLButtonElement>("manualBtn").onclick=()=>void open();
 el<HTMLButtonElement>("manualClose").onclick=()=>dialog.close();
 search.oninput=()=>void searchNow();
 search.onkeydown=e=>{if(!search.value.trim())return;const hits=Array.from(nav.querySelectorAll<HTMLButtonElement>("[data-entry]"));if(!hits.length)return;if(e.key==="ArrowDown"||e.key==="ArrowUp"){e.preventDefault();keyboardIndex=Math.max(0,Math.min(hits.length-1,keyboardIndex+(e.key==="ArrowDown"?1:-1)));hits.forEach((b,i)=>b.classList.toggle("keyboard-hit",i===keyboardIndex));hits[keyboardIndex].scrollIntoView({block:"nearest"})}if(e.key==="Enter"){e.preventDefault();hits[Math.max(0,keyboardIndex)]?.click()}};
 const grip=el("manualGrip"),layout=grip.parentElement!;let width=240;
 function resize(w:number){width=Math.max(150,Math.min(420,w));layout.style.setProperty("--manual-width",width+"px");grip.setAttribute("aria-valuenow",String(width))}
 grip.onkeydown=e=>{if(e.key==="ArrowLeft"||e.key==="ArrowRight"){e.preventDefault();resize(width+(e.key==="ArrowRight"?12:-12))}};
 grip.onpointerdown=e=>{grip.setPointerCapture(e.pointerId);const x=e.clientX,start=width;grip.onpointermove=ev=>resize(start+ev.clientX-x);grip.onlostpointercapture=()=>{grip.onpointermove=null}};
 dialog.addEventListener("keydown",e=>{if((e.ctrlKey||e.metaKey)&&e.key.toLowerCase()==="f"){e.preventDefault();search.focus();search.select()}});
 dialog.onclose=()=>{if(model)saved.set(model.fingerprint,{id:current,key:currentKey,scroll:detail.scrollTop,navScroll:nav.scrollTop,expanded:[...openSections]});generation++;queryEpoch++;model=undefined;entries.clear()};
 return {open,isOpen:()=>dialog.open}
}

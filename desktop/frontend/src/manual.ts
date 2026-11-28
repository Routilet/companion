type Variant={key:string;label:string;symbols:string[];signatures:string[];advancedSignatures?:string[];note:string;example:string};
type Entry={tags?:string[];keywords?:string[];summary?:string;parameters?:{name:string;description:string}[];returns?:string;requirements?:string[];errors?:string[];complexity?:string;id:string;title:string;description:string;example:string;signatures:string[];commonSignatures?:string[];advancedSignatures?:string[];related?:string[];variants?:Variant[]};
type Section={title:string;children?:Section[];entries?:string[]};
type Reference={entries?:Entry[];tags?:string[];version:string;notes:string;categories:{title:string;entries:Entry[]}[];sections:Section[]};
type Manual={reference:Reference;origin:string;sourcePath?:string;fingerprint:string};
type Hit={id:string;variant?:string;category:string};
type Call=<T>(method:string,...args:unknown[])=>Promise<T>;
const el=<T extends HTMLElement=HTMLElement>(id:string)=>document.getElementById(id) as T;
const node=(tag:string,text="",cls="")=>{const n=document.createElement(tag);n.textContent=text;if(cls)n.className=cls;return n};
const button=(text:string,action:()=>void)=>{const b=document.createElement("button");b.textContent=text;b.type="button";b.className="ui-button";b.onclick=action;return b};
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
 const dialog=el("manualDialog"),search=el<HTMLInputElement>("manualSearch"),nav=el("manualNav"),detail=el("manualDetail"),status=el("manualStatus"),version=el("manualVersion");
 let model:Manual|undefined,entries=new Map<string,Entry>(),generation=0,queryEpoch=0,current="",currentKey="",trail:{id:string;key:string}[]=[];
 const saved=new Map<string,{id:string;key:string;scroll:number;navScroll:number;query:string;tags:string[]}>();let selectedTags=new Set<string>();let keyboardIndex=-1;
 const filter=el<HTMLDetailsElement>("manualFilter"),tagPanel=el("manualTags"),chosen=el("manualSelectedTags"),count=el("manualCount");
 function toggleTag(t:string){if(selectedTags.has(t))selectedTags.delete(t);else selectedTags.add(t);renderTags();void searchNow()}
 function renderTags(){
  const tags=[...new Set([...entries.values()].flatMap(e=>e.tags??[]))].sort((a,b)=>a.localeCompare(b,"zh-CN"));
  tagPanel.replaceChildren();chosen.replaceChildren();filter.hidden=!tags.length;
  filter.querySelector("summary")!.textContent=selectedTags.size?selectedTags.size+" 个标签":"全部标签";
  for(const t of tags){const b=button(t,()=>toggleTag(t));b.setAttribute("aria-pressed",String(selectedTags.has(t)));tagPanel.append(b)}
  for(const t of selectedTags){const b=button(t+" ×",()=>toggleTag(t));b.setAttribute("aria-label","移除标签 "+t);chosen.append(b)}
  chosen.hidden=!selectedTags.size;
 }
 function matches(e:Entry){return [...selectedTags].every(t=>e.tags?.includes(t))}
 function symbolNames(e:Entry){const names=e.variants?.flatMap(v=>v.symbols)??[];const common=e.signatures.map(s=>s.match(/([A-Za-z_]\\w*)\\s*\\(/)?.[1]??s.replace(/^typedef.*?([A-Za-z_]\\w*);$/,"$1"));return [...new Set([...names,...common])].join(" · ")}

 function setStatus(s:string){status.textContent=s}
 function clear(){nav.replaceChildren();detail.replaceChildren();current="";currentKey="";trail=[]}
 function highlight(){for(const b of Array.from(nav.querySelectorAll<HTMLButtonElement>("[data-entry]")))b.setAttribute("aria-current",String(b.dataset.entry===current))}
 function locate(){const b=Array.from(nav.querySelectorAll<HTMLButtonElement>("[data-entry]")).find(b=>b.dataset.entry===current);b?.scrollIntoView({block:"nearest"})}
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
  head.append(node("h2",e.title));detail.append(head);
  if(e.summary)detail.append(node("p",e.summary));
  if(e.tags?.length){const row=node("div","","manual-tags manual-detail-tags");for(const t of e.tags)row.append(button(t,()=>{if(!selectedTags.has(t))toggleTag(t)}));detail.append(row)}
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
  if(e.parameters?.length){detail.append(node("h3","参数"));const table=node("table","","manual-params");for(const p of e.parameters){const tr=node("tr"),name=node("td");name.append(node("code",p.name));tr.append(name,node("td",p.description));table.append(tr)}detail.append(table)}
  if(e.returns){detail.append(node("h3","返回值"));prose(detail,e.returns)}
  if(e.requirements?.length){detail.append(node("h3","使用条件"));for(const s of e.requirements)prose(detail,s)}
  if(e.errors?.length){detail.append(node("h3","失败行为"));for(const s of e.errors)prose(detail,s)}
  if(e.description&&e.description!==e.summary){detail.append(node("h3","使用说明"));prose(detail,e.description)}
  if(v?.note)prose(detail,v.note);if(e.complexity)detail.append(node("p","复杂度："+e.complexity,"muted"));
  const example=v?.example??e.example;
  if(example){
   const row=node("div","","manualHead");row.append(node("h3","调用示例"));
   const fp=model.fingerprint,selectedKey=currentKey;
   const copy=button("复制示例",()=>{copy.disabled=true;void call("CopyManualExample",id,selectedKey,fp).then(()=>{copy.textContent="已复制";setTimeout(()=>{copy.textContent="复制示例"},1200)}).catch(err=>setStatus(String(err))).finally(()=>{copy.disabled=false})});
   row.append(copy);detail.append(row,codeBlock(example))
  }
  if(advanced.length){const more=document.createElement("details");more.append(node("summary","扩展接口 · 显式容量"),codeBlock(advanced.join("\n")));detail.append(more)}
  if(e.related?.length){detail.append(node("h3","相关操作"));const related=node("div","","manualVariants");for(const id of e.related){const target=entries.get(id);if(target)related.append(button(target.title,()=>show(id,"",true)))}detail.append(related)}
  detail.scrollTop=0
 }
 function entryButton(id:string,key="",category=""){
  const e=entries.get(id);if(!e)return node("span",id);
  const b=button(e.title,()=>{trail=[];show(id,key)});b.dataset.entry=id;b.classList.add("ui-list-item");b.title=symbolNames(e);
  const v=e.variants?.find(v=>v.key===key);b.append(node("small",v?.symbols.join(" · ")??symbolNames(e)));return b
 }
 function renderList(hits?:Hit[]){
  const list=hits?hits.filter(h=>{const e=entries.get(h.id);return !!e&&matches(e)}):[...entries.values()].filter(matches).map(e=>({id:e.id,variant:"",category:""}));
  nav.replaceChildren();keyboardIndex=-1;count.textContent=list.length+" 项";
  for(const h of list)nav.append(entryButton(h.id,h.variant,h.category));highlight();
  if(!list.length)nav.append(node("p","没有匹配的函数","muted"));
 }
 async function searchNow(){
  const epoch=++queryEpoch,session=generation,q=search.value.trim();
  if(!model)return;
  if(!q){renderList();return}
  try{const hits=await call<Hit[]>("SearchManual",q);if(epoch!==queryEpoch||session!==generation||dialog.hidden)return;renderList(hits)}
  catch(e){if(epoch===queryEpoch&&session===generation)setStatus(String(e))}
 }
 async function open(){
  if(!dialog.hidden||!canOpen())return;
  const session=++generation;queryEpoch++;search.disabled=true;setStatus("正在读取本地资源…");dialog.hidden=false;el("workspace").hidden=true;el("workbenchSidebar").hidden=true;el("manualSidebar").hidden=false;el("manualBtn").setAttribute("aria-pressed","true");el("workbenchBtn").setAttribute("aria-pressed","false");
  try{const m=await call<Manual>("OpenManual");if(session!==generation||dialog.hidden)return;
   if(model?.fingerprint===m.fingerprint){search.disabled=false;setStatus("");const previous=saved.get(m.fingerprint);if(search.value.trim())await searchNow();if(session===generation&&!dialog.hidden&&previous){detail.scrollTop=previous.scroll;nav.scrollTop=previous.navScroll}return}model=m;clear();search.value="";const source=m.reference.entries??m.reference.categories.flatMap(c=>c.entries.map(e=>({...e,tags:e.tags??[c.title]})));
   entries=new Map(source.map(e=>[e.id,e] as const));
   version.textContent="Exam Kit "+m.reference.version+(m.origin==="snapshot"?" · 当前文件资源快照":" · 已安装资源")+" · "+entries.size+" 项";
   const previous=saved.get(m.fingerprint);selectedTags=new Set(previous?.tags??[]);search.value=previous?.query??"";renderTags();renderList();if(search.value.trim())void searchNow();if(previous?.id&&entries.has(previous.id)){show(previous.id,previous.key);detail.scrollTop=previous.scroll;nav.scrollTop=previous.navScroll}else notes();search.disabled=false;setStatus("");search.focus()
  }catch(e){if(session===generation&&!dialog.hidden){model=undefined;entries.clear();clear();search.value="";version.textContent="";setStatus(String(e));detail.append(node("p","无法读取匹配的手册。请检查当前文件的资源快照，或在 Routilet 中检查 Exam Kit 安装状态。"))}}
 }
 el<HTMLButtonElement>("manualBtn").onclick=()=>void open();
 function close(){if(dialog.hidden)return;filter.open=false;if(model)saved.set(model.fingerprint,{id:current,key:currentKey,scroll:detail.scrollTop,navScroll:nav.scrollTop,query:search.value,tags:[...selectedTags]});generation++;queryEpoch++;dialog.hidden=true;el("workspace").hidden=false;el("workbenchSidebar").hidden=false;el("manualSidebar").hidden=true;el("manualBtn").setAttribute("aria-pressed","false");el("workbenchBtn").setAttribute("aria-pressed","true")}
 el<HTMLButtonElement>("workbenchBtn").onclick=close;
 el<HTMLButtonElement>("manualNotes").onclick=notes;
 search.oninput=()=>void searchNow();
 search.onkeydown=e=>{if(e.key==="Escape"){e.preventDefault();if(filter.open)filter.open=false;else{search.value="";void searchNow()}return}const hits=Array.from(nav.querySelectorAll<HTMLButtonElement>("[data-entry]"));if(!hits.length)return;if(e.key==="ArrowDown"||e.key==="ArrowUp"){e.preventDefault();keyboardIndex=Math.max(0,Math.min(hits.length-1,keyboardIndex+(e.key==="ArrowDown"?1:-1)));hits.forEach((b,i)=>b.classList.toggle("keyboard-hit",i===keyboardIndex));hits[keyboardIndex].scrollIntoView({block:"nearest"})}if(e.key==="Enter"){e.preventDefault();hits[Math.max(0,keyboardIndex)]?.click()}};
 document.addEventListener("keydown",e=>{if(!dialog.hidden&&(e.ctrlKey||e.metaKey)&&e.key.toLowerCase()==="f"){e.preventDefault();search.focus();search.select()}});
 
 return {open,close,isOpen:()=>!dialog.hidden}
}

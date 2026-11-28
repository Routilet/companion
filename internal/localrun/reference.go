package localrun

import (
 "bytes"
 "encoding/json"
 "errors"
 "fmt"
 "strings"
 "sort"
 "regexp"
)

type ReferenceVariant struct {
 Key string `json:"key"`
 Label string `json:"label"`
 Symbols []string `json:"symbols"`
 Signatures []string `json:"signatures"`
 AdvancedSignatures []string `json:"advancedSignatures,omitempty"`
 Note string `json:"note"`
 Example string `json:"example"`
}
type ReferenceParameter struct { Name string `json:"name"`; Description string `json:"description"` }
type ReferenceEntry struct {
 Tags []string `json:"tags,omitempty"`
 Summary string `json:"summary,omitempty"`
 Parameters []ReferenceParameter `json:"parameters,omitempty"`
 Returns string `json:"returns,omitempty"`
 Requirements []string `json:"requirements,omitempty"`
 Errors []string `json:"errors,omitempty"`
 Complexity string `json:"complexity,omitempty"`
 ID string `json:"id"`
 Title string `json:"title"`
 Description string `json:"description"`
 Example string `json:"example"`
 Keywords []string `json:"keywords,omitempty"`
 Related []string `json:"related,omitempty"`
 Signatures []string `json:"signatures"`
 CommonSignatures []string `json:"commonSignatures,omitempty"`
 AdvancedSignatures []string `json:"advancedSignatures,omitempty"`
 Variants []ReferenceVariant `json:"variants,omitempty"`
}
type ReferenceCategory struct {
 Title string `json:"title"`
 Entries []ReferenceEntry `json:"entries"`
}
type ReferenceSection struct {
 Title string `json:"title"`
 Children []ReferenceSection `json:"children,omitempty"`
 Entries []string `json:"entries,omitempty"`
}
type Reference struct {
 Tags []string `json:"tags,omitempty"`
 Entries []ReferenceEntry `json:"entries,omitempty"`
 SchemaVersion int `json:"schemaVersion"`
 Version string `json:"version"`
 Notes string `json:"notes"`
 Categories []ReferenceCategory `json:"categories"`
 Sections []ReferenceSection `json:"sections"`
}
type Manual struct {
 Reference Reference `json:"reference"`
 Origin string `json:"origin"`
 SourcePath string `json:"sourcePath,omitempty"`
 Fingerprint string `json:"fingerprint"`
}
func ParseReference(raw []byte,version string)(Reference,error){
 var r Reference
 if e:=json.Unmarshal(raw,&r);e!=nil{return r,e}
 if r.Version!=version{return r,errors.New("函数手册版本无效")}
 if r.SchemaVersion==2{return validateFlat(r)}
 if r.SchemaVersion!=1||len(r.Categories)==0{return r,errors.New("函数手册格式或版本无效")}
 ids:=map[string]ReferenceEntry{}
 for _,c:=range r.Categories{
  if strings.TrimSpace(c.Title)==""||len(c.Entries)==0{return r,errors.New("函数手册分类无效")}
  for _,e:=range c.Entries{
   if e.ID==""||e.Title==""||e.Description==""||len(e.Signatures)==0||ids[e.ID].ID!=""{return r,fmt.Errorf("函数手册条目无效或重复: %s",e.ID)}
   ids[e.ID]=e
   keys:=map[string]bool{}
   for _,v:=range e.Variants{
    if v.Key==""||v.Label==""||keys[v.Key]||len(v.Symbols)==0||len(v.Signatures)==0{return r,fmt.Errorf("函数手册类型变体无效: %s",e.ID)}
    keys[v.Key]=true
   }
  }
 }
 for _,e:=range ids{for _,id:=range e.Related{if id==e.ID||ids[id].ID==""{return r,fmt.Errorf("相关条目引用无效: %s",id)}}}
 seen:=map[string]bool{}
 var visit func([]ReferenceSection,int)error
 visit=func(nodes []ReferenceSection,depth int)error{
  if depth>8||len(nodes)==0{return errors.New("函数手册目录无效")}
  for _,n:=range nodes{
   if n.Title==""||(len(n.Children)>0)==(len(n.Entries)>0){return errors.New("函数手册目录节点无效")}
   if len(n.Children)>0{if e:=visit(n.Children,depth+1);e!=nil{return e}}
   for _,id:=range n.Entries{if ids[id].ID==""{return fmt.Errorf("未知手册条目: %s",id)};seen[id]=true}
  };return nil
 }
 if e:=visit(r.Sections,0);e!=nil{return r,e}
 if len(seen)!=len(ids){return r,errors.New("函数手册目录缺少条目")}
 r.Entries=nil;r.Tags=nil
 paths:=map[string][]string{}
 var collect func([]ReferenceSection,[]string)
 collect=func(nodes []ReferenceSection,parents []string){for _,n:=range nodes{tags:=append(append([]string{},parents...),n.Title);for _,id:=range n.Entries{paths[id]=tags};collect(n.Children,tags)}}
 collect(r.Sections,nil)
 for _,c:=range r.Categories{for _,e:=range c.Entries{e.Tags=paths[e.ID];if len(e.Tags)==0{e.Tags=[]string{c.Title}};r.Entries=append(r.Entries,e)}}
 return r,nil
}

// Read native flat schema v2; historical v1 resources remain readable.
func validateFlat(r Reference)(Reference,error){
 if r.SchemaVersion!=2||len(r.Entries)==0||len(r.Entries)>4096||len(r.Tags)==0||len(r.Tags)>128{return r,errors.New("函数手册 v2 格式无效")}
 tags:=map[string]bool{};for _,tag:=range r.Tags{if strings.TrimSpace(tag)==""||tags[tag]{return r,errors.New("重复或空手册标签")};tags[tag]=true}
 ids:=map[string]bool{}
 for _,e:=range r.Entries{
  if !kitID.MatchString(e.ID)||ids[e.ID]||strings.TrimSpace(e.Title)==""||strings.TrimSpace(e.Summary)==""||len(e.Signatures)==0||len(e.Tags)==0{return r,fmt.Errorf("函数手册 v2 条目无效: %s",e.ID)}
  ids[e.ID]=true;used:=map[string]bool{}
  for _,tag:=range e.Tags{if !tags[tag]||used[tag]{return r,fmt.Errorf("无效手册标签: %s",tag)};used[tag]=true}
  keys:=map[string]bool{};for _,v:=range e.Variants{if v.Key==""||v.Label==""||keys[v.Key]||len(v.Symbols)==0||len(v.Signatures)==0{return r,errors.New("无效函数变体")};keys[v.Key]=true}
  params:=map[string]bool{};for _,p:=range e.Parameters{if !kitID.MatchString(p.Name)||strings.TrimSpace(p.Description)==""||params[p.Name]{return r,errors.New("无效参数说明")};params[p.Name]=true}
 }
 for _,e:=range r.Entries{for _,id:=range e.Related{if id==e.ID||!ids[id]{return r,fmt.Errorf("无效相关条目: %s",id)}}}
 return r,nil
}
func(r Reference) AllEntries()[]ReferenceEntry{
 if len(r.Entries)>0{return r.Entries}
 var out []ReferenceEntry
 for _,c:=range r.Categories{for _,e:=range c.Entries{if len(e.Tags)==0{e.Tags=[]string{c.Title}};out=append(out,e)}}
 return out
}
// Reference selection is independent of successful assembly/compilation.
// A student can consult the correct manual even while editing broken C.
func manualKit(path,packageRoot string)(Kit,string,error){
 if path!=""{
  s,e:=Open(path);if e!=nil{return Kit{},"",e}
  if !isCPP(path){
   raw,e:=readSource(s);if e!=nil{return Kit{},"",e}
   scan:=bytes.ReplaceAll(bytes.ReplaceAll(raw,[]byte("\\\r\n"),nil),[]byte("\\\n"),nil)
   name:=""
   for _,line:=range sourceLines(maskC(scan,false)){
    m:=ppDirective.FindStringSubmatch(strings.TrimRight(string(line),"\r\n"))
    if m==nil||m[1]!="include"{continue}
    inc:=literalInclude.FindStringSubmatch(m[2])
    if inc==nil{
     if strings.Contains(m[2],".companion/resources/"){return Kit{},"",errors.New("当前源码的 Exam Kit 引用不完整，请检查头文件路径")}
     continue
    }
    if inc[2]!=""&&strings.Contains(strings.ReplaceAll(inc[2],"\\","/"),".companion/resources/"){
     if name!=""{return Kit{},"",errors.New("当前源码存在多个 Exam Kit 引用")}
     name=inc[2]
    }
   }
   if name!=""{k,e:=kitFromInclude(s,name);return k,"snapshot",e}
  }
 }
 k,e:=LoadKitPackage(packageRoot);return k,"installed",e
}
func LoadSnapshotManual(path string)(Manual,bool,error){
 k,origin,e:=manualKit(path,"");if origin!="snapshot"{return Manual{},false,nil};if e!=nil{return Manual{},true,e};r,e:=ParseReference(k.Files["reference.json"],k.Version);if e!=nil{return Manual{},true,e};return Manual{Reference:r,Origin:origin,Fingerprint:k.Fingerprint,SourcePath:path},true,nil
}
func LoadManual(path,packageRoot string)(Manual,error){
 k,origin,e:=manualKit(path,packageRoot);if e!=nil{return Manual{},e}
 r,e:=ParseReference(k.Files["reference.json"],k.Version);if e!=nil{return Manual{},e}
 m:=Manual{Reference:r,Origin:origin,Fingerprint:k.Fingerprint}
 if origin=="snapshot"{m.SourcePath=path}
 return m,nil
}
func(m Manual) Example(id,key string)(string,error){
 for _,e:=range m.Reference.AllEntries(){if e.ID!=id{continue}
  example:=e.Example
  if len(e.Variants)>0{
   found:=false
   for _,v:=range e.Variants{if v.Key==key{example=v.Example;found=true;break}}
   if !found{return "",errors.New("未知手册类型变体")}
  }else if key!=""{return "",errors.New("条目没有类型变体")}
  if example==""{return "",errors.New("此条目没有调用示例")}
  return example,nil
 }
 return "",errors.New("未知手册条目")
}
type ReferenceHit struct{
 ID string `json:"id"`
 Variant string `json:"variant,omitempty"`
 Category string `json:"category"`
 Score int `json:"-"`
}
// Search is local, deterministic, and matches exact function names before
// partial names, Chinese titles, and descriptive text.
var referenceSymbol=regexp.MustCompile(`([A-Za-z_]\w*)\s*\(`)
func SearchReference(r Reference,q string)[]ReferenceHit{
 q=strings.ToLower(strings.TrimSpace(q));hits:=[]ReferenceHit{}
 if q==""{return hits}
 terms:=strings.Fields(q)
 for _,e:=range r.AllEntries(){
  hay:=[]string{e.ID,e.Title,e.Summary,e.Description,e.Returns,e.Complexity}
  hay=append(hay,e.Tags...);hay=append(hay,e.Signatures...);hay=append(hay,e.Keywords...);hay=append(hay,e.Requirements...);hay=append(hay,e.Errors...)
  for _,p:=range e.Parameters{hay=append(hay,p.Name,p.Description)}
  score:=5;key:=""
  if strings.ToLower(e.ID)==q{score=0}
  match:=func(s,v string){s=strings.ToLower(s);if s==q{if score>0||key==""{score=0;key=v}}else if strings.Contains(s,q)&&score>1{score=1;key=v}}
  for _,v:=range e.Variants{hay=append(hay,v.Symbols...);hay=append(hay,v.Label,v.Note);for _,s:=range v.Symbols{match(s,v.Key)}}
  for _,s:=range e.Signatures{if m:=referenceSymbol.FindStringSubmatch(s);len(m)>1{match(m[1],"")}}
  if score>2&&strings.Contains(strings.ToLower(e.Title),q){score=2}
  for _,s:=range e.Keywords{if score>3&&strings.Contains(strings.ToLower(s),q){score=3}}
  for _,s:=range e.Tags{if score>4&&strings.Contains(strings.ToLower(s),q){score=4}}
  body:=strings.ToLower(strings.Join(hay," "));ok:=true
  for _,t:=range terms{if !strings.Contains(body,t){ok=false;break}}
  if ok{hits=append(hits,ReferenceHit{e.ID,key,strings.Join(e.Tags," · "),score})}
 }
 sort.SliceStable(hits,func(i,j int)bool{return hits[i].Score<hits[j].Score})
 return hits
}

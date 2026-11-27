package localrun

import (
 "bytes"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "os"
 "path/filepath"
 "regexp"
 "strings"
 "unicode/utf8"
)

// Exam Kit is data, not a runtime or executable dependency.
// 04a deliberately consumes only the actual schema 2 package.
const KitPackageName = "@routilet/c-exam-kit"
const KitPreviewEnv = "COMPANION_EXAM_KIT_PACKAGE"

type KitComponent struct {
 ID string `json:"id"`
 Symbols []string `json:"symbols"`
 Requires []string `json:"requires"`
 Code string `json:"code"`
}
type KitManifest struct {
 SchemaVersion int `json:"schemaVersion"`
 ID string `json:"id"`
 Language string `json:"language"`
 LanguageID string `json:"languageId"`
 Standard string `json:"standard"`
 HeaderFile string `json:"headerFile"`
 SystemHeaders []string `json:"systemHeaders"`
 Components []KitComponent `json:"components"`
}
type Kit struct {
 Version string
 Manifest KitManifest
 Order []string
 Files map[string][]byte
 Fingerprint string
}
type KitStatus struct {
 Version string `json:"version"`
 Path string `json:"path"`
 Error string `json:"error,omitempty"`
}
type kitSeal struct {
 SchemaVersion int `json:"schemaVersion"`
 Version string `json:"version"`
 Fingerprint string `json:"fingerprint"`
}
var kitVersion=regexp.MustCompile("^[0-9]+\\.[0-9]+\\.[0-9]+(-[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?$")
var kitID=regexp.MustCompile("^[A-Za-z_][A-Za-z_0-9]*$")
var kitHeader=regexp.MustCompile("^[A-Za-z_][A-Za-z_0-9.-]*\\.h$")
var kitSystemHeader=regexp.MustCompile("^[A-Za-z0-9_./+-]+\\.h$")
var kitInclude=regexp.MustCompile("(?m)^[ \\t]*#[ \\t]*include\\b")
var kitGuardChar=regexp.MustCompile("[^A-Za-z0-9]")

func readKitFile(dir,name string)([]byte,error){
 p:=filepath.Join(dir,name)
 st,e:=os.Lstat(p);if e!=nil{return nil,e}
 if !st.Mode().IsRegular(){return nil,fmt.Errorf("Exam Kit 文件不是普通文件: %s",name)}
 f,e:=os.Open(p);if e!=nil{return nil,e};defer f.Close()
 b,e:=io.ReadAll(io.LimitReader(f,16*1024*1024+1));if e!=nil{return nil,e}
 if len(b)>16*1024*1024||!utf8.Valid(b){return nil,fmt.Errorf("Exam Kit 文件过大或编码无效: %s",name)}
 return b,nil
}
func parseKitManifest(raw []byte)(KitManifest,[]string,error){
 var m KitManifest
 if e:=json.Unmarshal(raw,&m);e!=nil{return m,nil,e}
 if m.SchemaVersion!=2||m.ID!="exam-io-c99"||m.Language!="c"||m.LanguageID!="c99"||m.Standard!="c99"||!kitHeader.MatchString(m.HeaderFile)||m.SystemHeaders==nil||len(m.Components)==0||len(m.Components)>4096{
  return m,nil,errors.New("Exam Kit manifest 不符合现行 C99 schema 2")
 }
 for _,h:=range m.SystemHeaders{if !kitSystemHeader.MatchString(h)||strings.Contains(h,"..")||strings.HasPrefix(h,"/"){return m,nil,errors.New("无效系统头文件: "+h)}}
 byID:=map[string]KitComponent{};symbols:=map[string]bool{}
 for _,c:=range m.Components{
  if !kitID.MatchString(c.ID)||byID[c.ID].ID!=""||c.Symbols==nil||c.Requires==nil||strings.TrimSpace(c.Code)==""||kitInclude.MatchString(c.Code){return m,nil,errors.New("无效或重复的组件: "+c.ID)}
  byID[c.ID]=c
  for _,s:=range c.Symbols{if !kitID.MatchString(s)||symbols[s]{return m,nil,errors.New("无效或重复公开符号: "+s)};symbols[s]=true}
  for _,dep:=range c.Requires{if !kitID.MatchString(dep){return m,nil,errors.New("无效依赖: "+dep)}}
 }
 state:=map[string]int{};order:=[]string{}
 var visit func(string)error
 visit=func(id string)error{
  c,ok:=byID[id];if !ok{return errors.New("未知组件依赖: "+id)}
  if state[id]==1{return errors.New("循环组件依赖: "+id)}
  if state[id]==2{return nil}
  state[id]=1
  for _,dep:=range c.Requires{if e:=visit(dep);e!=nil{return e}}
  state[id]=2;order=append(order,id);return nil
 }
 for _,c:=range m.Components{if e:=visit(c.ID);e!=nil{return m,nil,e}}
 return m,order,nil
}
// RenderKitHeader deliberately matches the old renderHeader algorithm, including
// the guard and whitespace. This is the complete IDE header, not submission code.
func RenderKitHeader(m KitManifest,order []string)[]byte{
 var b strings.Builder
 guard:="ACCODING_"+strings.ToUpper(kitGuardChar.ReplaceAllString(m.HeaderFile,"_"))
 fmt.Fprintf(&b,"#ifndef %s\n#define %s\n",guard,guard)
 for _,h:=range m.SystemHeaders{fmt.Fprintf(&b,"#include <%s>\n",h)}
 byID:=map[string]string{}
 for _,c:=range m.Components{byID[c.ID]=c.Code}
 for _,id:=range order{code:=byID[id];b.WriteByte('\n');b.WriteString(code);if !strings.HasSuffix(code,"\n"){b.WriteByte('\n')}}
 b.WriteString("\n#endif\n")
 return []byte(b.String())
}
func validateReference(raw []byte,version string)error{
 var ref struct{
  SchemaVersion int
  Version string
  Categories []json.RawMessage
  Sections []json.RawMessage
 }
 if e:=json.Unmarshal(raw,&ref);e!=nil{return e}
 if ref.SchemaVersion!=1||ref.Version!=version||len(ref.Categories)==0||len(ref.Sections)==0{return errors.New("Exam Kit reference.json 格式或版本不匹配")}
 return nil
}
func kitFingerprint(files map[string][]byte)string{
 // encoding/json sorts string map keys; hashing bytes preserves exact contents.
 b,_:=json.Marshal(files);return digest(b)
}
func readKitData(dir,version string)(Kit,error){
 k:=Kit{Version:version,Files:map[string][]byte{}}
 if len(version)>100||!kitVersion.MatchString(version){return k,errors.New("无效 Exam Kit 版本")}
 if e:=safeDir(dir,false);e!=nil{return k,e}
 for _,name:=range []string{"manifest.json","reference.json","CC-BY-4.0.txt"}{
  b,e:=readKitFile(dir,name);if e!=nil{return k,fmt.Errorf("读取 %s: %w",name,e)}
  if len(b)==0{return k,errors.New("资源为空: "+name)};k.Files[name]=b
 }
 if b,e:=readKitFile(dir,"manual.zh-CN.md");e==nil{k.Files["manual.zh-CN.md"]=b}else if !errors.Is(e,os.ErrNotExist){return k,e}
 m,order,e:=parseKitManifest(k.Files["manifest.json"]);if e!=nil{return k,e}
 if e=validateReference(k.Files["reference.json"],version);e!=nil{return k,e}
 k.Manifest=m;k.Order=order;k.Files["expanded.h"]=RenderKitHeader(m,order)
 return k,nil
}
func LoadKitPackage(packageRoot string)(Kit,error){
 if packageRoot==""{return Kit{},errors.New("未找到 Exam Kit；请先准备已安装资源包")}
 if !filepath.IsAbs(packageRoot){return Kit{},errors.New("Exam Kit 安装目录必须是绝对路径")}
 root,e:=filepath.EvalSymlinks(packageRoot);if e!=nil{return Kit{},e}
 if e=safeDir(root,false);e!=nil{return Kit{},e}
 raw,e:=readKitFile(root,"package.json");if e!=nil{return Kit{},e}
 var pkg struct{Name,Version string}
 if e=json.Unmarshal(raw,&pkg);e!=nil{return Kit{},e}
 if pkg.Name!=KitPackageName{return Kit{},errors.New("不是当前 Exam Kit 资源包")}
 k,e:=readKitData(filepath.Join(root,"preset"),pkg.Version);if e!=nil{return k,e}
 license,e:=readKitFile(root,"LICENSE");if e!=nil{return k,e}
 if len(license)==0{return k,errors.New("Exam Kit LICENSE 为空")}
 k.Files["LICENSE"]=license;k.Fingerprint=kitFingerprint(k.Files)
 return k,nil
}
func LoadKitSnapshot(dir string)(Kit,error){
 var k Kit
 if e:=safeDir(dir,false);e!=nil{return k,e}
 b,e:=readKitFile(dir,"snapshot.json");if e!=nil{return k,e}
 var seal kitSeal
 if e=json.Unmarshal(b,&seal);e!=nil{return k,e}
 if seal.SchemaVersion!=1||seal.Version!=filepath.Base(dir){return k,errors.New("Exam Kit 快照版本不匹配")}
 k,e=readKitData(dir,seal.Version);if e!=nil{return k,e}
 header,e:=readKitFile(dir,"expanded.h");if e!=nil{return k,e}
 if !bytes.Equal(header,k.Files["expanded.h"]){return k,errors.New("Exam Kit 完整头文件与 manifest 不一致")}
 k.Files["LICENSE"],e=readKitFile(dir,"LICENSE");if e!=nil{return k,e}
 k.Fingerprint=kitFingerprint(k.Files)
 if k.Fingerprint!=seal.Fingerprint{return k,errors.New("Exam Kit 快照内容已修改；请核对原版本")}
 return k,nil
}
func SaveKitSnapshot(root string,k Kit)(string,error){
 if len(k.Version)>100||!kitVersion.MatchString(k.Version)||k.Fingerprint!=kitFingerprint(k.Files){return "",errors.New("无效资源版本或内容")}
 if e:=safeDir(root,false);e!=nil{return "",e}
 meta:=filepath.Join(root,".companion")
 if e:=safeDir(meta,true);e!=nil{return "",e};hideDirectory(meta)
 resources:=filepath.Join(meta,"resources")
 if e:=safeDir(resources,true);e!=nil{return "",e}
 unlock,e:=lockCases(filepath.Join(resources,".write.lock"));if e!=nil{return "",e};defer unlock()
 dest:=filepath.Join(resources,k.Version)
 if _,e=os.Lstat(dest);e==nil{
  existing,e:=LoadKitSnapshot(dest);if e!=nil{return "",e}
  if existing.Fingerprint!=k.Fingerprint{return "",errors.New("同版本 Exam Kit 资源内容不一致；不覆盖已有快照")}
  return dest,nil
 }else if !errors.Is(e,os.ErrNotExist){return "",e}
 stage,e:=os.MkdirTemp(resources,".stage-");if e!=nil{return "",e};defer os.RemoveAll(stage)
 for name,data:=range k.Files{
  switch name{case "manifest.json","reference.json","manual.zh-CN.md","expanded.h","LICENSE","CC-BY-4.0.txt":default:return "",errors.New("未知资源文件: "+name)}
  if e=os.WriteFile(filepath.Join(stage,name),data,0600);e!=nil{return "",e}
 }
 seal,e:=json.MarshalIndent(kitSeal{1,k.Version,k.Fingerprint},"","  ");if e!=nil{return "",e}
 if e=os.WriteFile(filepath.Join(stage,"snapshot.json"),append(seal,'\n'),0600);e!=nil{return "",e}
 // The lock serializes Companion writers; an external writer need not honor it.
 if e=absent(dest);e!=nil{return "",e}
 if e=os.Rename(stage,dest);e!=nil{return "",e}
 return dest,nil
}

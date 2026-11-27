package localrun

import (
 "bytes"
 "errors"
 "fmt"
 "path/filepath"
 "regexp"
 "strings"
 "unicode/utf8"
)

type LineOrigin struct {
 File string
 Line int
}
type PreparedSource struct {
 Bytes []byte
 SourceDigest string
 PreparedDigest string
 ResourceDigest string
 KitVersion string
 Components []string
 Origins []LineOrigin
}
var ppDirective=regexp.MustCompile("^[ \\t\\v\\f]*#[ \\t\\v\\f]*([A-Za-z_][A-Za-z_0-9]*)(.*)$")
var literalInclude=regexp.MustCompile("^[ \\t\\v\\f]*(?:<([^>\\r\\n]+)>|\"([^\"\\r\\n]+)\")[ \\t\\v\\f\\r]*$")
var safeSystemName=regexp.MustCompile("^[A-Za-z0-9_+./-]+$")
var identifier=regexp.MustCompile("[A-Za-z_][A-Za-z_0-9]*")
var functionDefinition=regexp.MustCompile("\\b([A-Za-z_][A-Za-z_0-9]*)[ \\t\\r\\n]*\\([^;{}]*\\)[ \\t\\r\\n]*\\{")

// maskC preserves byte offsets, line boundaries, and token separation.
// For assembly, comments are replaced by spaces rather than concatenating tokens.
// For identifier selection, quoted character/string literals are masked too.
func maskC(src []byte,literals bool)[]byte{
 out:=append([]byte(nil),src...)
 blank:=func(i int){if src[i]!='\n'&&src[i]!='\r'{out[i]=' '}}
 for i:=0;i<len(src);{
  if i+1<len(src)&&src[i]=='/'&&src[i+1]=='/'{
   for i<len(src)&&src[i]!='\n'{blank(i);i++}
   continue
  }
  if i+1<len(src)&&src[i]=='/'&&src[i+1]=='*'{
   blank(i);blank(i+1);i+=2
   for i<len(src){
    if i+1<len(src)&&src[i]=='*'&&src[i+1]=='/'{blank(i);blank(i+1);i+=2;break}
    blank(i);i++
   }
   continue
  }
  if src[i]=='"'||src[i]=='\''{
   quote:=src[i];if literals{blank(i)};i++
   for i<len(src){
    ch:=src[i];if literals{blank(i)};i++
    if ch=='\\'&&i<len(src){if literals{blank(i)};i++;continue}
    if ch==quote||ch=='\n'{break}
   }
   continue
  }
  i++
 }
 return out
}

func sourceLines(b []byte)[][]byte{
 if len(b)==0{return nil}
 lines:=bytes.SplitAfter(b,[]byte{'\n'})
 if len(lines[len(lines)-1])==0{lines=lines[:len(lines)-1]}
 return lines
}

// Kit snapshots may live in the source directory or an ancestor. No
// arbitrary local header or symlink traversal is treated as a Kit reference.
func kitFromInclude(s Source,name string)(Kit,error){
 if filepath.IsAbs(name)||filepath.VolumeName(name)!=""||strings.ContainsAny(name,"\\:\r\n"){return Kit{},errors.New("Exam Kit 必须使用相对 include 路径")}
 base,e:=filepath.EvalSymlinks(s.Directory);if e!=nil{return Kit{},e}
 header:=filepath.Clean(filepath.Join(base,filepath.FromSlash(name)))
 rel,e:=filepath.Rel(base,header);if e!=nil||filepath.ToSlash(rel)!=name{return Kit{},errors.New("Exam Kit include 不是规范的相对路径")}
 versionDir:=filepath.Dir(header);resources:=filepath.Dir(versionDir);meta:=filepath.Dir(resources);root:=filepath.Dir(meta)
 if filepath.Base(header)!="expanded.h"||filepath.Base(resources)!="resources"||filepath.Base(meta)!=".companion"||!kitVersion.MatchString(filepath.Base(versionDir)){
  return Kit{},errors.New("仅支持 .companion/resources/<version>/expanded.h 本地引用")
 }
 down,e:=filepath.Rel(root,base)
 if e!=nil||filepath.IsAbs(down)||down==".."||strings.HasPrefix(down,".."+string(filepath.Separator)){return Kit{},errors.New("Exam Kit 必须位于源码目录或其上级目录")}
 for _,p:=range []string{root,meta,resources,versionDir}{if e=safeDir(p,false);e!=nil{return Kit{},fmt.Errorf("Exam Kit 路径不可用: %w",e)}}
 k,e:=LoadKitSnapshot(versionDir);if e!=nil{return k,fmt.Errorf("读取 Exam Kit %s 快照失败: %w",filepath.Base(versionDir),e)}
 return k,nil
}

func PrepareSource(s Source,raw []byte)(PreparedSource,error){
 p:=PreparedSource{Bytes:raw,SourceDigest:digest(raw),PreparedDigest:digest(raw)}
 if isCPP(s.Path){if e:=checkSource(raw);e!=nil{return p,e};return p,nil}
 // Inspect translation-phase line splices for ordinary C, but retain the exact
 // bytes for compilation. Kit assembly intentionally rejects line splices
 // (including trigraph spellings) instead of pretending to preprocess them.
 scan:=bytes.ReplaceAll(bytes.ReplaceAll(raw,[]byte("\\\r\n"),nil),[]byte("\\\n"),nil)
 comments:=maskC(scan,false)
 lines:=sourceLines(comments)
 includeAt:=-1
 local:=""
 for i,line:=range lines{
  m:=ppDirective.FindStringSubmatch(strings.TrimRight(string(line),"\r\n"))
  if m==nil||m[1]!="include"{continue}
  inc:=literalInclude.FindStringSubmatch(m[2]);if inc==nil{return p,fmt.Errorf("第 %d 行：不支持宏生成或复杂 include",i+1)}
  if inc[1]!=""{
   if !safeSystemName.MatchString(inc[1])||strings.Contains(inc[1],"..")||strings.HasPrefix(inc[1],"/"){return p,fmt.Errorf("第 %d 行：不支持该系统头路径",i+1)}
   continue
  }
  if includeAt>=0{return p,errors.New("仅支持恰好一次 Exam Kit 本地头文件引用")}
  includeAt=i;local=inc[2]
 }
 // C digraph/trigraph directives could hide a local include from this limited
 // scanner. Reject them conservatively outside comments/literals.
 tokens:=maskC(scan,true)
 if bytes.Contains(tokens,[]byte("%:"))||bytes.Contains(tokens,[]byte("??="))||bytes.Contains(raw,[]byte("??/")){
  return p,errors.New("独立源码模式不支持替代预处理符号或三字符转义")
 }
 if includeAt<0{return p,nil}
 if !utf8.Valid(raw){return p,errors.New("含 Exam Kit 的源码必须为 UTF-8")}
 if !bytes.Equal(scan,raw){return p,errors.New("含 Exam Kit 的源码暂不支持反斜杠续行")}
 for i,line:=range lines{
  trim:=strings.TrimSpace(string(line))
  if strings.HasPrefix(trim,"#"){
   m:=ppDirective.FindStringSubmatch(strings.TrimRight(string(line),"\r\n"))
   if m==nil||m[1]!="include"{return p,fmt.Errorf("第 %d 行：含 Exam Kit 的源码仅支持 #include 预处理指令",i+1)}
  }
 }
 k,e:=kitFromInclude(s,local);if e!=nil{return p,e}
 byID:=map[string]KitComponent{};used:=map[string]bool{};defs:=map[string]bool{};selected:=map[string]bool{}
 var user strings.Builder
 for _,line:=range sourceLines(tokens){
  if strings.HasPrefix(strings.TrimSpace(string(line)),"#"){user.WriteByte('\n');continue}
  user.Write(line)
 }
 for _,word:=range identifier.FindAllString(user.String(),-1){used[word]=true}
 for _,m:=range functionDefinition.FindAllStringSubmatch(user.String(),-1){defs[m[1]]=true}
 for _,c:=range k.Manifest.Components{byID[c.ID]=c}
 var selectID func(string)
 selectID=func(id string){if selected[id]{return};selected[id]=true;for _,dep:=range byID[id].Requires{selectID(dep)}}
 for _,c:=range k.Manifest.Components{
  for _,sym:=range c.Symbols{
   if defs[sym]{return p,errors.New("源码定义了 Exam Kit 保留函数: "+sym)}
   if used[sym]{selectID(c.ID)}
  }
 }
 // Private component definitions are reserved only if selected as dependencies.
 for id:=range selected{if defs[id]{return p,errors.New("源码定义与 Exam Kit 依赖冲突: "+id)}}
 var out bytes.Buffer
 origins:=[]LineOrigin{}
 write:=func(data []byte,file string,start int){
  out.Write(data);for i:=range sourceLines(data){origins=append(origins,LineOrigin{file,start+i})}
 }
 // The replacement happens at the include's original position, so earlier
 // user declarations and later source retain order. All generated segments
 // end in newline before the next segment, enabling exact line mapping.
 for i,line:=range lines{
  if i!=includeAt{write(line,s.Path,i+1);continue}
  var headers strings.Builder
  for _,h:=range k.Manifest.SystemHeaders{fmt.Fprintf(&headers,"#include <%s>\n",h)}
  write([]byte(headers.String()),"Exam Kit "+k.Version+" / headers",1)
  for _,id:=range k.Order{
   if !selected[id]{continue}
   p.Components=append(p.Components,id)
   write([]byte("\n"),"Exam Kit "+k.Version+" / "+id,0)
   code:=[]byte(byID[id].Code);if !bytes.HasSuffix(code,[]byte("\n")){code=append(code,'\n')}
   write(maskC(code,false),"Exam Kit "+k.Version+" / "+id,1)
  }
  write([]byte("\n"),s.Path,i+1)
 }
 p.Bytes=out.Bytes();p.PreparedDigest=digest(p.Bytes);p.ResourceDigest=k.Fingerprint;p.KitVersion=k.Version;p.Origins=origins
 return p,nil
}
func PreparePath(path string)(Source,PreparedSource,error){
 s,e:=Open(path);if e!=nil{return s,PreparedSource{},e}
 raw,e:=readSource(s);if e!=nil{return s,PreparedSource{},e}
 p,e:=PrepareSource(s,raw);return s,p,e
}
func mapPreparedDiagnostics(ds []Diagnostic,p PreparedSource,source string){
 for i:=range ds{
  d:=&ds[i]
  if d.File==source&&d.Line>0&&d.Line<=len(p.Origins){
   origin:=p.Origins[d.Line-1]
   d.File=origin.File;d.Line=origin.Line
   // User and generated comments are replaced by equal byte widths, but
   // display columns still depend on compiler Unicode/tab conventions.
  }
 }
}

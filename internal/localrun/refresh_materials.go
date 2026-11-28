package localrun

import (
 "encoding/json"
 "errors"
 "os"
 "path/filepath"
 "reflect"
 "strings"
 "unicode/utf8"
)

// RefreshProblemData replaces explicitly confirmed OJ materials only. A
// restorable copy of each changed local document/case set is made first.
// The caller must supply the snapshot captured before the network read.
func RefreshProblemData(root,source string, previous *Document, cases CaseSet, markdown string, samples []Case) (bool,string,error) {
 if _,e:=Within(root,source);e!=nil{return false,"",e}
 current,e:=ReadDocument(root,source);if e!=nil{return false,"",e}
 if !reflect.DeepEqual(current,previous){return false,"",errors.New("题面已在外部修改，请重新更新")}
 now,e:=LoadCases(source);if e!=nil{return false,"",e}
 if now.Revision!=cases.Revision{return false,"",errors.New("样例已在外部修改，请重新更新")}
 docChanged:=markdown!=""&&(previous==nil||previous.Content!=markdown)
 casesChanged:=len(samples)>0&&!reflect.DeepEqual(cases.Cases,samples)
 if !docChanged&&!casesChanged{return false,"",nil}
 if docChanged&&(len(markdown)>1<<20||!utf8.ValidString(markdown)){return false,"",errors.New("题面无效或过大")}
 if casesChanged{if e=validateCases(samples);e!=nil{return false,"",e}}
 // Backups are scoped to each source, with unique names; existing backups
 // are never overwritten. No source file is written.
 dir:=filepath.Join(filepath.Dir(source),".companion")
 if e=safeDir(dir,true);e!=nil{return false,"",e};hideDirectory(dir)
 dir=filepath.Join(dir,"refresh-backups")
 if e=safeDir(dir,true);e!=nil{return false,"",e}
 backup,e:=os.MkdirTemp(dir,filepath.Base(source)+"-");if e!=nil{return false,"",e}
 if docChanged&&previous!=nil{
  if e=os.WriteFile(filepath.Join(backup,previous.Name),[]byte(previous.Content),0600);e!=nil{return false,backup,e}
 }
 if casesChanged&&cases.Revision!="missing"{
  raw,err:=json.MarshalIndent(caseDocument{SchemaVersion:1,Cases:cases.Cases},"","  ");if err!=nil{return false,backup,err}
  if e=os.WriteFile(filepath.Join(backup,"cases.json"),append(raw,'\n'),0600);e!=nil{return false,backup,e}
 }
 // SaveCases has its own optimistic revision check and atomic replacement.
 changed:=false
 if casesChanged{
  if _,e=SaveCases(source,samples,cases.Revision);e!=nil{return false,backup,e};changed=true
 }
 if docChanged{
  if _,e=Within(root,source);e!=nil{return changed,backup,e}
  current,e=ReadDocument(root,source);if e!=nil{return changed,backup,e}
  if !reflect.DeepEqual(current,previous){return changed,backup,errors.New("题面已在外部修改，未覆盖")}
  if previous==nil{
   // New documents use a source-specific name to avoid sharing a problem.md.
   e=SaveDocument(root,source,filepath.Base(source)+".md",markdown)
  }else{
   name:=previous.Name
   if name!=filepath.Base(source)+".md"&&name!="problem.md"&&name!=strings.TrimSuffix(filepath.Base(source),filepath.Ext(source))+".md"{return changed,backup,errors.New("题面文件名无效")}
   path:=filepath.Join(filepath.Dir(source),name)
   if _,e=Within(root,path);e!=nil{return changed,backup,e}
   f,err:=os.CreateTemp(filepath.Dir(path),".document-refresh-*");if err!=nil{return changed,backup,err}
   defer os.Remove(f.Name())
   _,e=f.WriteString(markdown);if e==nil{e=f.Sync()};ce:=f.Close();if e==nil{e=ce}
   if e==nil{
    current,e=ReadDocument(root,source)
    if e==nil&&!reflect.DeepEqual(current,previous){e=errors.New("题面已在外部修改，未覆盖")}
    if e==nil{e=replaceCases(f.Name(),path)}
   }
  }
  if e!=nil{return changed,backup,e};changed=true
 }
 return changed,backup,nil
}

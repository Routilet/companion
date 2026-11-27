package localrun

import (
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "sort"
 "strings"
)

type Entry struct {
 Path string `json:"path"`
 Name string `json:"name"`
 Directory bool `json:"directory"`
}
func sourceExtension(p string)bool{
 switch strings.ToLower(filepath.Ext(p)){case ".c",".cpp",".cc",".cxx":return true};return false
}
func isCPP(p string)bool{return !strings.EqualFold(filepath.Ext(p),".c")}
func hiddenDir(n string)bool{return strings.HasPrefix(n,".")||strings.EqualFold(n,"node_modules")}
func OpenDirectory(p string)(string,error){
 p,e:=filepath.Abs(p);if e!=nil{return "",e}
 if e=safeDir(p,false);e!=nil{return "",e}
 return filepath.EvalSymlinks(p)
}
// Within validates every directory component below the canonical selected root.
// It never follows a link in the navigation tree.
func Within(root,path string)(string,error){
 if root==""{return "",errors.New("请先打开文件夹")}
 p,e:=filepath.Abs(path);if e!=nil{return "",e}
 rel,e:=filepath.Rel(root,p)
 if e!=nil||rel==".."||strings.HasPrefix(rel,".."+string(filepath.Separator))||filepath.IsAbs(rel){return "",errors.New("路径不在当前文件夹内")}
 if e=safeDir(root,false);e!=nil{return "",e}
 if rel=="."{return p,nil}
 parts:=strings.Split(rel,string(filepath.Separator));current:=root
 for i,n:=range parts{
  if hiddenDir(n){return "",errors.New("不浏览隐藏辅助目录")}
  current=filepath.Join(current,n)
  st,e:=os.Lstat(current);if e!=nil{return "",e}
  if st.Mode()&os.ModeSymlink!=0||(!st.IsDir()&&!st.Mode().IsRegular()){return "",errors.New("不跟随链接或特殊文件")}
  if i<len(parts)-1&&!st.IsDir(){return "",errors.New("无效目录")}
 }
 return p,nil
}
func ListDirectory(root,dir string)([]Entry,error){
 p,e:=Within(root,dir);if e!=nil{return nil,e}
 if e=safeDir(p,false);e!=nil{return nil,e}
 list,e:=os.ReadDir(p);if e!=nil{return nil,e}
 entries:=[]Entry{}
 for _,v:=range list{
  if hiddenDir(v.Name()){continue}
  st,e:=v.Info();if e!=nil{continue}
  if st.Mode()&os.ModeSymlink!=0{continue}
  if !st.IsDir()&&(!st.Mode().IsRegular()||!sourceExtension(v.Name())){continue}
  entries=append(entries,Entry{filepath.Join(p,v.Name()),v.Name(),st.IsDir()})
 }
 sort.SliceStable(entries,func(i,j int)bool{if entries[i].Directory!=entries[j].Directory{return entries[i].Directory};return strings.ToLower(entries[i].Name)<strings.ToLower(entries[j].Name)})
 return entries,nil
}
func CreateFolder(root,dir,name string)(string,error){
 // Reuse the cross-platform source-name validation for the basename.
 if !validName(name+".c")||strings.HasSuffix(name,".")||name!=strings.TrimSpace(name){return "",errors.New("文件夹名称无效")}
 p,e:=Within(root,dir);if e!=nil{return "",e};if e=safeDir(p,false);e!=nil{return "",e}
 target:=filepath.Join(p,name);if e=os.Mkdir(target,0700);e!=nil{return "",e};return target,nil
}
func validName(n string)bool{
 if n==""||n!=strings.TrimSpace(n)||strings.HasSuffix(n,".")||strings.HasPrefix(n,".")||strings.ContainsAny(n,"/\\<>:\"|?*\x00\r\n")||!sourceExtension(n){return false}
 base:=strings.ToUpper(strings.Split(n,".")[0])
 switch base{case "CON","PRN","AUX","NUL","COM1","COM2","COM3","COM4","COM5","COM6","COM7","COM8","COM9","LPT1","LPT2","LPT3","LPT4","LPT5","LPT6","LPT7","LPT8","LPT9":return false}
 return true
}
func absent(p string)error{
 _,e:=os.Lstat(p);if e==nil{return fmt.Errorf("目标已存在：%s",p)}
 if !errors.Is(e,os.ErrNotExist){return e};return nil
}
func CreateSource(root,dir,name string)(Source,error){return CreateSourceWithKit(root,dir,name,os.Getenv(KitPreviewEnv))}
func CreateSourceWithKit(root,dir,name,packageRoot string)(Source,error){
 if !validName(name){return Source{},errors.New("请输入有效的 C/C++ 文件名")}
 p,e:=Within(root,dir);if e!=nil{return Source{},e}
 if e=safeDir(p,false);e!=nil{return Source{},e}
 target:=filepath.Join(p,name)
 if e=absent(target);e!=nil{return Source{},e}
 // Never attach abandoned test data to a newly created program.
 cp:=filepath.Join(p,".companion","cases",name+".json")
 if e=caseDirs(cp,false);e==nil{if e=absent(cp);e!=nil{return Source{},e}}else if !errors.Is(e,os.ErrNotExist){return Source{},e}
 code:="#include <stdio.h>\n\nint main(void) {\n    return 0;\n}\n"
 if isCPP(name){code="#include <iostream>\n\nint main() {\n    return 0;\n}\n"}else{
  k,e:=LoadKitPackage(packageRoot);if e!=nil{return Source{},e}
  snapshot,e:=SaveKitSnapshot(root,k);if e!=nil{return Source{},e}
  include,e:=filepath.Rel(p,filepath.Join(snapshot,"expanded.h"));if e!=nil{return Source{},e}
  include=filepath.ToSlash(include);if strings.ContainsAny(include,"\"\r\n"){return Source{},errors.New("资源相对路径不能表示为 C include")}
  code="#include \""+include+"\"\n"+code
 }
 f,e:=os.OpenFile(target,os.O_CREATE|os.O_EXCL|os.O_WRONLY,0600);if e!=nil{return Source{},e}
 _,e=f.WriteString(code);closeErr:=f.Close()
 if e==nil{e=closeErr};if e!=nil{return Source{},fmt.Errorf("新建未完整写入，请检查 %s: %w",target,e)}
 return Open(target)
}
// linkMove is intentionally no-clobber, including concurrent destination creation.
// On filesystems without hard links it returns an error rather than overwriting.
func linkMove(from,to string)error{
 if e:=os.Link(from,to);e!=nil{return e}
 if e:=os.Remove(from);e!=nil{
  if cleanup:=os.Remove(to);cleanup!=nil{return fmt.Errorf("移动未完成，两处路径均保留：%s；%s (%v; %v)",from,to,e,cleanup)}
  return e
 }
 return nil
}
func RenameSource(root,path,name string)(Source,error){
 if !validName(name){return Source{},errors.New("文件名无效")}
 p,e:=Within(root,path);if e!=nil{return Source{},e}
 s,e:=Open(p);if e!=nil{return Source{},e}
 if !strings.EqualFold(filepath.Ext(name),filepath.Ext(s.Name)){return Source{},errors.New("重命名不更改源文件类型")}
 if name==s.Name{return s,nil}
 target:=filepath.Join(s.Directory,name)
 if e=absent(target);e!=nil{return Source{},e}
 before,_:=casePath(s.Path);after:=filepath.Join(filepath.Dir(before),name+".json")
 exists:=false
 if e=caseDirs(before,false);e==nil{
  release,e:=lockCases(filepath.Join(filepath.Dir(before),".write.lock"));if e!=nil{return Source{},e};defer release()
  if e=absent(after);e!=nil{return Source{},e}
  st,e:=os.Lstat(before);if e==nil{if !st.Mode().IsRegular(){return Source{},errors.New("旧样例不是普通文件")};exists=true}else if !errors.Is(e,os.ErrNotExist){return Source{},e}
 }else if !errors.Is(e,os.ErrNotExist){return Source{},e}
 if e=linkMove(s.Path,target);e!=nil{return Source{},e}
 if exists{
  if e=linkMove(before,after);e!=nil{
   if rollback:=linkMove(target,s.Path);rollback!=nil{return Source{},fmt.Errorf("样例移动失败，源码回滚失败；请检查 %s 和 %s；旧样例 %s 保留 (%v; %v)",s.Path,target,before,e,rollback)}
   return Source{},fmt.Errorf("样例移动失败，源码已回滚：%w",e)
  }
 }
 return Open(target)
}
type Orphan struct {
 Name string `json:"name"`
 Count int `json:"count"`
}
func ListOrphans(source string)([]Orphan,error){
 p,e:=casePath(source);if e!=nil{return nil,e}
 out:=[]Orphan{}
 if e=caseDirs(p,false);e!=nil{if errors.Is(e,os.ErrNotExist){return out,nil};return nil,e}
 list,e:=os.ReadDir(filepath.Dir(p));if e!=nil{return nil,e}
 for _,v:=range list{
  n:=strings.TrimSuffix(v.Name(),".json");if n==v.Name()||!validName(n){continue}
  if e=absent(filepath.Join(filepath.Dir(source),n));e!=nil{continue}
  cs,e:=readCaseDocument(filepath.Join(filepath.Dir(p),v.Name()));if e!=nil||cs.Revision=="missing"{continue}
  out=append(out,Orphan{n,len(cs.Cases)})
 }
 return out,nil
}
func ImportCases(source,jsonPath,revision string,appendExisting bool)(CaseSet,error){
 target,e:=LoadCases(source);if e!=nil{return CaseSet{},e}
 if target.Revision!=revision{return CaseSet{},errors.New("当前样例已变化，请重新读取")}
 if len(target.Cases)>0&&!appendExisting{return CaseSet{},errors.New("已有样例；需明确追加")}
 incoming,e:=readCaseDocument(jsonPath);if e!=nil{return CaseSet{},e}
 if incoming.Revision=="missing"{return CaseSet{},errors.New("导入文件不存在")}
 return SaveCases(source,append(target.Cases,incoming.Cases...),revision)
}
func RestoreOrphan(source,name,revision string)(CaseSet,error){
 if !validName(name){return CaseSet{},errors.New("无效样例名称")}
 list,e:=ListOrphans(source);if e!=nil{return CaseSet{},e}
 for _,o:=range list{if o.Name==name{
  p,_:=casePath(source);return ImportCases(source,filepath.Join(filepath.Dir(p),name+".json"),revision,false)
 }}
 return CaseSet{},errors.New("未找到未关联的样例")
}

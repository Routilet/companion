package localrun
import("errors";"io";"os";"path/filepath";"strings";"unicode/utf8")
type Document struct{Name string `json:"name"`;Content string `json:"content"`}
func ReadDocument(root,source string)(*Document,error){
 if _,e:=Within(root,source);e!=nil{return nil,e}
 dir:=filepath.Dir(source)
 for _,name:=range []string{filepath.Base(source)+".md","problem.md",strings.TrimSuffix(filepath.Base(source),filepath.Ext(source))+".md"}{
  p:=filepath.Join(dir,name);st,e:=os.Lstat(p);if os.IsNotExist(e){continue};if e!=nil{return nil,e};if !st.Mode().IsRegular()||st.Size()>1<<20{return nil,errors.New("题面不是普通文件或过大")};if _,e=Within(root,p);e!=nil{return nil,e}
  f,e:=os.Open(p);if e!=nil{return nil,e};raw,e:=io.ReadAll(io.LimitReader(f,(1<<20)+1));f.Close();if e!=nil||len(raw)>1<<20||!utf8.Valid(raw){return nil,errors.New("无法读取 UTF-8 文档")}
  return &Document{name,string(raw)},nil
 }
 return nil,nil
}
func SaveDocument(root,source,name,content string)error{
 if _,e:=Within(root,source);e!=nil{return e}
 if name!="problem.md"&&name!=filepath.Base(source)+".md"{return errors.New("无效题面文件名")}
 if content==""||len(content)>1<<20||!utf8.ValidString(content){return errors.New("题面无效或过大")}
 dir:=filepath.Dir(source);f,e:=os.CreateTemp(dir,".document-*");if e!=nil{return e};defer os.Remove(f.Name());_,e=f.WriteString(content);if e==nil{e=f.Sync()};ce:=f.Close();if e==nil{e=ce};if e!=nil{return e};return os.Link(f.Name(),filepath.Join(dir,name))
}

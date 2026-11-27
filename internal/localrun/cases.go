package localrun

import (
 "bytes"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "os"
 "path/filepath"
 "unicode/utf8"
)

const maxCases = 32
const maxCaseFile = 8 * 1024 * 1024

type Case struct {
 Input string `json:"input"`
 Expected *string `json:"expected"`
}
type CaseSet struct {
 Cases []Case `json:"cases"`
 Revision string `json:"revision"`
}
type caseDocument struct {
 SchemaVersion int `json:"schemaVersion"`
 Cases []Case `json:"cases"`
}
func casePath(source string)(string,error){
 s,e:=Open(source);if e!=nil{return "",e}
 return filepath.Join(s.Directory,".companion","cases",s.Name+".json"),nil
}
func validateCases(cases []Case)error{
 if len(cases)>maxCases{return fmt.Errorf("最多保存 %d 组测试",maxCases)}
 for _,c:=range cases{
  if len(c.Input)>maxInput||!utf8.ValidString(c.Input){return errors.New("输入超过 1 MiB 或不是有效 UTF-8")}
  if c.Expected!=nil&&(len(*c.Expected)>maxInput||!utf8.ValidString(*c.Expected)){return errors.New("预期输出超过 1 MiB 或不是有效 UTF-8")}
 }
 return nil
}
func safeDir(p string,create bool)error{
 st,e:=os.Lstat(p)
 if errors.Is(e,os.ErrNotExist)&&create{
  if e=os.Mkdir(p,0700);e!=nil&&!errors.Is(e,os.ErrExist){return e}
  st,e=os.Lstat(p)
 }
 if e!=nil{return e}
 if !st.IsDir()||st.Mode()&os.ModeSymlink!=0{return errors.New("辅助目录不是普通目录: "+p)}
 return nil
}
func caseDirs(p string,create bool)error{
 root:=filepath.Dir(filepath.Dir(p))
 if e:=safeDir(root,create);e!=nil{return e}
 if create{hideDirectory(root)}
 return safeDir(filepath.Dir(p),create)
}
func readCases(p string)(CaseSet,error){
 empty:=CaseSet{Cases:[]Case{},Revision:"missing"}
 if e:=caseDirs(p,false);e!=nil{if errors.Is(e,os.ErrNotExist){return empty,nil};return CaseSet{},e}
 return readCaseDocument(p)
}
func readCaseDocument(p string)(CaseSet,error){
 empty:=CaseSet{Cases:[]Case{},Revision:"missing"}
 st,e:=os.Lstat(p);if errors.Is(e,os.ErrNotExist){return empty,nil};if e!=nil{return CaseSet{},e}
 if !st.Mode().IsRegular(){return CaseSet{},errors.New("测试数据不是普通文件")}
 f,e:=os.Open(p);if e!=nil{return CaseSet{},e};defer f.Close()
 b,e:=io.ReadAll(io.LimitReader(f,maxCaseFile+1));if e!=nil{return CaseSet{},e}
 if len(b)>maxCaseFile||!utf8.Valid(b){return CaseSet{},errors.New("测试数据过大或编码无效")}
 var raw struct{SchemaVersion int;Cases []map[string]json.RawMessage}
 if e=json.Unmarshal(b,&raw);e!=nil||raw.SchemaVersion!=1||raw.Cases==nil{return CaseSet{},errors.New("测试数据损坏或版本不支持；原文件未修改")}
 cs:=make([]Case,0,len(raw.Cases))
 for _,v:=range raw.Cases{
  var c Case
  a,ok:=v["input"];if !ok||len(a)==0||a[0]!='"'||json.Unmarshal(a,&c.Input)!=nil{return CaseSet{},errors.New("测试数据 input 无效")}
  w,ok:=v["expected"];if !ok||json.Unmarshal(w,&c.Expected)!=nil{return CaseSet{},errors.New("测试数据 expected 无效")}
  cs=append(cs,c)
 }
 if e=validateCases(cs);e!=nil{return CaseSet{},e}
 return CaseSet{Cases:cs,Revision:digest(b)},nil
}
func LoadCases(source string)(CaseSet,error){p,e:=casePath(source);if e!=nil{return CaseSet{},e};return readCases(p)}
func SaveCases(source string,cases []Case,revision string)(CaseSet,error){
 if e:=validateCases(cases);e!=nil{return CaseSet{},e}
 if cases==nil{cases=[]Case{}}
 b,e:=json.MarshalIndent(caseDocument{SchemaVersion:1,Cases:cases},"","  ");if e!=nil{return CaseSet{},e}
 b=append(b,'\n');if len(b)>maxCaseFile{return CaseSet{},errors.New("测试数据文件超过 8 MiB")}
 p,e:=casePath(source);if e!=nil{return CaseSet{},e}
 if e=caseDirs(p,true);e!=nil{return CaseSet{},e}
 release,e:=lockCases(filepath.Join(filepath.Dir(p),".write.lock"));if e!=nil{return CaseSet{},e};defer release()
 previous,e:=readCases(p);if e!=nil{return CaseSet{},e}
 if previous.Revision!=revision{return CaseSet{},errors.New("测试数据已在外部修改，请重新读取；未覆盖原数据")}
 tmp,e:=os.CreateTemp(filepath.Dir(p),".save-*");if e!=nil{return CaseSet{},e}
 defer os.Remove(tmp.Name())
 if _,e=io.Copy(tmp,bytes.NewReader(b));e!=nil{tmp.Close();return CaseSet{},e}
 if e=tmp.Sync();e!=nil{tmp.Close();return CaseSet{},e}
 if e=tmp.Close();e!=nil{return CaseSet{},e}
 if e=replaceCases(tmp.Name(),p);e!=nil{return CaseSet{},e}
 return CaseSet{Cases:cases,Revision:digest(b)},nil
}

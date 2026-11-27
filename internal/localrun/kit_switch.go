package localrun
import("bytes";"errors";"os";"path/filepath";"strings";"unicode/utf8")
type KitChange struct{Path string `json:"path"`;Version string `json:"version"`;Kind string `json:"kind"`;Reason string `json:"reason,omitempty"`;Digest string `json:"-"`;next []byte}
type KitPlan struct{Version string `json:"version"`;Rows []KitChange `json:"rows"`;Fingerprint string `json:"-"`}
// Only the exact generated header line is editable; reject ambiguous sources.
// All other bytes (including comments and line endings) are retained.
func planKitFile(root,path string,k Kit)(KitChange,error){
 row:=KitChange{Path:path,Kind:"skip"}
 if _,e:=Within(root,path);e!=nil{return row,e}
 s,e:=Open(path);if e!=nil{return row,e};if isCPP(path){return row,errors.New("仅支持 C 源码")}
 raw,e:=readSource(s);if e!=nil{return row,e};row.Digest=digest(raw)
 if !utf8.Valid(raw)||bytes.Contains(raw,[]byte("\\\n"))||bytes.Contains(raw,[]byte("\\\r\n")){return row,errors.New("复杂编码或续行不支持自动切换")}
 rel,e:=filepath.Rel(s.Directory,filepath.Join(root,".companion","resources",k.Version,"expanded.h"));if e!=nil{return row,e}
 inc:=filepath.ToSlash(rel);if strings.ContainsAny(inc,"\"\r\n"){return row,errors.New("无法表示资源引用")}
 newLine:=[]byte("#include \""+inc+"\"")
 masked:=sourceLines(maskC(raw,false));lines:=sourceLines(raw);if len(masked)!=len(lines){return row,errors.New("无法识别源码")}
 oldLine:=-1;oldName:=""
 for i,line:=range masked{m:=ppDirective.FindStringSubmatch(strings.TrimRight(string(line),"\r\n"));if m==nil||m[1]!="include"{continue};v:=literalInclude.FindStringSubmatch(m[2]);if v==nil{return row,errors.New("复杂 include 不支持切换")};if v[2]!=""{if oldLine>=0{return row,errors.New("存在多个本地头文件")};oldLine=i;oldName=v[2]}}
 body:=[]byte("#include <stdio.h>\n\nint main(void) {\n    return 0;\n}\n")
 if oldLine<0{if bytes.Contains(raw,[]byte(".companion/resources/")){return row,errors.New("存在无法识别的资源引用")};row.Kind="attach";row.next=append(append(newLine,'\n'),raw...);return row,nil}
 old,e:=kitFromInclude(s,oldName);if e!=nil{return row,e};row.Version=old.Version
 if old.Version==k.Version{if old.Fingerprint!=k.Fingerprint{return row,errors.New("同版本内容不一致")};row.Kind="current";return row,nil}
 prefix:=[]byte("#include \""+oldName+"\"")
 if !bytes.Equal(bytes.TrimRight(lines[oldLine],"\r\n"),prefix){return row,errors.New("非标准资源引用，请手动修改")}
 exact:=append(append(append([]byte{},prefix...), '\n'),body...)
 row.Kind="modified";if bytes.Equal(raw,exact){row.Kind="template"}
 var buf bytes.Buffer;for i,line:=range lines{if i==oldLine{buf.Write(newLine);buf.Write(line[len(prefix):])}else{buf.Write(line)}};row.next=buf.Bytes();return row,nil
}
func PlanKitSwitch(root string,paths []string,k Kit)(KitPlan,error){
 p:=KitPlan{Version:k.Version,Fingerprint:k.Fingerprint,Rows:[]KitChange{}};if len(paths)==0||len(paths)>512{return p,errors.New("请选择 1 至 512 个 C 文件")}
 seen:=map[string]bool{};total:=0;for _,path:=range paths{if seen[path]{continue};seen[path]=true;r,e:=planKitFile(root,path,k);if e!=nil{r.Kind="skip";r.Reason=e.Error()};total+=len(r.next);if total>16<<20{return p,errors.New("源码总量过大，请缩小切换范围")};p.Rows=append(p.Rows,r)};return p,nil
}
func ApplyKitChange(root string,r KitChange,k Kit)error{
 if r.Kind!="template"&&r.Kind!="modified"&&r.Kind!="attach"{return errors.New("该文件不可切换")}
 fresh,e:=planKitFile(root,r.Path,k);if e!=nil{return e};if fresh.Digest!=r.Digest||fresh.Kind!=r.Kind||!bytes.Equal(fresh.next,r.next){return errors.New("源码已变化，请重新预览")}
 if _,e=SaveKitSnapshot(root,k);e!=nil{return e}
 s,e:=Open(r.Path);if e!=nil{return e};if _,e=PrepareSource(s,r.next);e!=nil{return errors.New("新资源不兼容当前源码："+e.Error())}
 // Recheck immediately before replacing. External editors should save and
 // close affected buffers before applying; their unsaved buffers are invisible.
 st,e:=os.Lstat(r.Path);if e!=nil||!st.Mode().IsRegular(){return errors.New("源码已变化")}
 tmp,e:=os.CreateTemp(filepath.Dir(r.Path),".kit-switch-*");if e!=nil{return e};defer os.Remove(tmp.Name())
 if e=tmp.Chmod(st.Mode().Perm());e==nil{_,e=tmp.Write(r.next)};if e==nil{e=tmp.Sync()};ce:=tmp.Close();if e==nil{e=ce};if e!=nil{return e}
 s,e=Open(r.Path);if e!=nil{return e};raw,e:=readSource(s);if e!=nil||digest(raw)!=r.Digest{return errors.New("源码已变化，请重新预览")}
 now,e:=os.Lstat(r.Path);if e!=nil||!os.SameFile(st,now){return errors.New("源码已被替换")}
 return replaceCases(tmp.Name(),r.Path)
}

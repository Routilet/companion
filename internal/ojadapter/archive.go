package ojadapter
import("errors";"os";"path/filepath";"strings";"unicode";"unicode/utf8")
// Validate existing ancestors without following links; missing ancestors are
// created individually by the desktop core, only after confirmation.
func archivePath(root,path string)error{
 rel,e:=filepath.Rel(root,path);if e!=nil||rel=="."||filepath.IsAbs(rel)||rel==".."||strings.HasPrefix(rel,".."+string(filepath.Separator)){return errors.New("导入路径不在目标目录内")}
 p:=root
 for i,n:=range strings.Split(rel,string(filepath.Separator)){if n==""||strings.HasPrefix(n,"."){return errors.New("导入路径无效")};p=filepath.Join(p,n);st,e:=os.Lstat(p);if os.IsNotExist(e){return nil};if e!=nil{return e};last:=i==len(strings.Split(rel,string(filepath.Separator)))-1;if st.Mode()&os.ModeSymlink!=0||(!last&&!st.IsDir())||(last&&!st.Mode().IsRegular()){return errors.New("导入路径存在链接或非普通文件")}}
 return nil
}
type CaseInspection func(string)(string,bool,error)
func BuildArchivePlan(result ImportResult,dir,ext,attach string,inspect CaseInspection)(Plan,error){return BuildArchivePlanNamed(result,dir,ext,attach,"",inspect)}
func BuildArchivePlanNamed(result ImportResult,dir,ext,attach,name string,inspect CaseInspection)(Plan,error){
 if e:=result.Validate(result.AccountID);e!=nil{return Plan{},e}
 if !filepath.IsAbs(dir)||ext!=".c"&&ext!=".cpp"{return Plan{},errors.New("导入目录或语言无效")}
 if attach!=""&&(len(result.Problems)!=1||filepath.Dir(attach)!=dir){return Plan{},errors.New("只允许单题关联当前文件")}
 out:=Plan{Title:result.Title,Directory:dir,AccountID:result.AccountID,Attach:attach,Archive:true,Rows:[]PlanRow{}}
 // A previous flat import with a matching binding stays flat; no automatic
 // migration or duplicate source creation. Other existing files are conflicts.
 legacy:=false
 for _,p:=range result.Problems{n,_:=p.Filename(ext);path:=filepath.Join(dir,n);if b,e:=LoadBinding(path);e==nil&&b!=nil&&b.Target==p.Target{legacy=true;break}}
 group:="";if !legacy&&attach==""&&len(result.Problems)>0&&result.Problems[0].Target.Kind=="contest"{var e error;group,out.CanRename,e=archiveGroup(result,dir,ext,name);if e!=nil{return Plan{},e};out.FolderName=group}
 for _,p:=range result.Problems{
  n,_:=p.Filename(ext);rel:=n;doc:=n+".md"
  if !legacy&&attach==""{folder:=strings.TrimSuffix(n,ext);if p.Target.Kind=="contest"{folder=filepath.Join(group,folder)};rel=filepath.Join(folder,n);doc="problem.md"}
  if attach!=""{rel=filepath.Base(attach);doc=rel+".md"}
  row:=PlanRow{Name:rel,Title:p.Title,Action:"create",Samples:len(p.Samples),Warnings:append([]string{},p.Warnings...),problem:p}
  if p.Statement!=""{row.Document=doc}
  path:=filepath.Join(dir,rel);e:=archivePath(dir,path)
  if e==nil{
   st,err:=os.Lstat(path)
   if err==nil{
    row.info=st;b,err:=LoadBinding(path)
    if err!=nil{e=err}else if b!=nil&&b.Target==p.Target{row.Action="repair"}else if b==nil&&attach!=""{row.Action="attach"}else{e=errors.New("源码已存在或关联不同题目")}
    if e==nil&&(!st.Mode().IsRegular()||!strings.EqualFold(filepath.Ext(path),ext)&&!(ext==".cpp"&&(strings.EqualFold(filepath.Ext(path),".cc")||strings.EqualFold(filepath.Ext(path),".cxx")))){e=errors.New("源码类型无效")}
    if e==nil{var empty bool;row.revision,empty,e=inspect(path);if !empty{row.Samples=0;row.Warnings=append(row.Warnings,"保留现有样例")}}
   }else if !os.IsNotExist(err)||attach!=""{e=errors.New("目标源码无效")}else{e=checkNew(path)}
  }
  if e==nil&&row.Document!=""&&row.Action!="create"{for _,name:=range []string{filepath.Base(path)+".md","problem.md",strings.TrimSuffix(filepath.Base(path),filepath.Ext(path))+".md"}{if st,err:=os.Lstat(filepath.Join(filepath.Dir(path),name));err==nil&&st.Mode().IsRegular(){row.Document="";row.Warnings=append(row.Warnings,"保留已有本地文档");break}}}
  if e==nil&&row.Document!=""{if st,err:=os.Lstat(filepath.Join(filepath.Dir(path),row.Document));err==nil{if row.Action=="create"||!st.Mode().IsRegular(){e=errors.New("题面路径冲突")}else{row.Document="";row.Warnings=append(row.Warnings,"保留已有题面")}}else if !os.IsNotExist(err){e=err}}
  if e==nil&&row.Action=="repair"&&row.Samples==0&&row.Document==""{e=errors.New("已有同题关联，无缺失资料")}
  if e!=nil{row.Action="skip";row.Samples=0;row.Document="";row.Warnings=append(row.Warnings,e.Error())}
  out.Rows=append(out.Rows,row)
 }
 return out,nil
}

func validArchiveName(name string)bool{
 if name==""||len(name)>160||!utf8.ValidString(name)||name!=strings.TrimSpace(name)||strings.HasPrefix(name,".")||strings.HasSuffix(name,".")||strings.ContainsAny(name,"/\\<>:\"|?*"){return false}
 for _,r:=range name{if unicode.IsControl(r){return false}}
 base:=strings.ToUpper(strings.TrimSpace(strings.SplitN(name,".",2)[0]))
 if base=="CON"||base=="PRN"||base=="AUX"||base=="NUL"{return false}
 if len(base)==4&&(strings.HasPrefix(base,"COM")||strings.HasPrefix(base,"LPT"))&&base[3]>='1'&&base[3]<='9'{return false};return true
}
func titleArchiveName(title string)string{
 name:=strings.Map(func(r rune)rune{if unicode.IsControl(r)||strings.ContainsRune("/\\<>:\"|?*",r){return '-'};return r},title)
 name=strings.Trim(strings.TrimSpace(name),". ");for len(name)>140{_,n:=utf8.DecodeLastRuneInString(name);name=name[:len(name)-n]};name=strings.TrimRight(name,". ")
 if name==""{return "比赛"};if !validArchiveName(name){return "比赛 "+name};return name
}
// Check actual bound sources, not folder names. Reusing a renamed/legacy
// archive preserves repairs; ambiguity fails closed rather than choosing a copy.
func archiveGroup(result ImportResult,dir,ext,requested string)(string,bool,error){
 expected:=map[string]Problem{};for _,p:=range result.Problems{n,e:=p.Filename(ext);if e!=nil{return "",false,e};expected[strings.TrimSuffix(n,ext)]=p}
 entries,e:=os.ReadDir(dir);if e!=nil{return "",false,e}
 scanned:=0
 matches:=func(base string)(bool,error){
  children,e:=os.ReadDir(base);if e!=nil{return false,e}
  for _,child:=range children{
   scanned++;if scanned>8192{return false,errors.New("目录过大，请选择更具体的导入位置")}
   p,ok:=expected[child.Name()];if !ok||!child.IsDir()||child.Type()&os.ModeSymlink!=0{continue}
   n,_:=p.Filename(ext);src:=filepath.Join(base,child.Name(),n)
   if e:=archivePath(dir,src);e!=nil{continue}
   st,e:=os.Lstat(src);if e!=nil||!st.Mode().IsRegular(){continue}
   b,e:=LoadBinding(src);if e==nil&&b!=nil&&b.Target==p.Target{return true,nil}
  };return false,nil
 }
 // If the user selected the archive itself, never add a second outer folder.
 if ok,e:=matches(dir);e!=nil{return "",false,e}else if ok{return "",false,nil}
 found:=[]string{}
 for _,entry:=range entries{if !entry.IsDir()||entry.Type()&os.ModeSymlink!=0||strings.HasPrefix(entry.Name(),".")||strings.EqualFold(entry.Name(),"node_modules")||strings.HasSuffix(strings.ToLower(entry.Name()),".dsym"){continue};if ok,e:=matches(filepath.Join(dir,entry.Name()));e!=nil{return "",false,e}else if ok{found=append(found,entry.Name())}}
 if len(found)>1{return "",false,errors.New("存在多个同一比赛的本地副本，请选择具体比赛目录")}
 if len(found)==1{return found[0],false,nil}
 name:=requested;if name==""{name=titleArchiveName(result.Title)}
 if !validArchiveName(name){return "",false,errors.New("比赛文件夹名称无效")}
 occupied:=func(n string)bool{for _,v:=range entries{if strings.EqualFold(v.Name(),n){return true}};return false}
 if occupied(name)&&requested==""{name=name+" ("+result.Problems[0].Target.ContestID+")"}
 if !validArchiveName(name)||occupied(name){return "",false,errors.New("比赛文件夹已存在，请选择其他名称或已有比赛目录")}
 return name,true,nil
}

package ojadapter
import("errors";"os";"path/filepath";"strings")
// Validate existing ancestors without following links; missing ancestors are
// created individually by the desktop core, only after confirmation.
func archivePath(root,path string)error{
 rel,e:=filepath.Rel(root,path);if e!=nil||rel=="."||filepath.IsAbs(rel)||rel==".."||strings.HasPrefix(rel,".."+string(filepath.Separator)){return errors.New("导入路径不在目标目录内")}
 p:=root
 for i,n:=range strings.Split(rel,string(filepath.Separator)){if n==""||strings.HasPrefix(n,"."){return errors.New("导入路径无效")};p=filepath.Join(p,n);st,e:=os.Lstat(p);if os.IsNotExist(e){return nil};if e!=nil{return e};last:=i==len(strings.Split(rel,string(filepath.Separator)))-1;if st.Mode()&os.ModeSymlink!=0||(!last&&!st.IsDir())||(last&&!st.Mode().IsRegular()){return errors.New("导入路径存在链接或非普通文件")}}
 return nil
}
type CaseInspection func(string)(string,bool,error)
func BuildArchivePlan(result ImportResult,dir,ext,attach string,inspect CaseInspection)(Plan,error){
 if e:=result.Validate(result.AccountID);e!=nil{return Plan{},e}
 if !filepath.IsAbs(dir)||ext!=".c"&&ext!=".cpp"{return Plan{},errors.New("导入目录或语言无效")}
 if attach!=""&&(len(result.Problems)!=1||filepath.Dir(attach)!=dir){return Plan{},errors.New("只允许单题关联当前文件")}
 out:=Plan{Title:result.Title,Directory:dir,AccountID:result.AccountID,Attach:attach,Archive:true,Rows:[]PlanRow{}}
 // A previous flat import with a matching binding stays flat; no automatic
 // migration or duplicate source creation. Other existing files are conflicts.
 legacy:=false
 for _,p:=range result.Problems{n,_:=p.Filename(ext);path:=filepath.Join(dir,n);if b,e:=LoadBinding(path);e==nil&&b!=nil&&b.Target==p.Target{legacy=true;break}}
 for _,p:=range result.Problems{
  n,_:=p.Filename(ext);rel:=n;doc:=n+".md"
  if !legacy&&attach==""{folder:=strings.TrimSuffix(n,ext);if p.Target.Kind=="contest"{folder=filepath.Join("contest-"+p.Target.ContestID,folder)};rel=filepath.Join(folder,n);doc="problem.md"}
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

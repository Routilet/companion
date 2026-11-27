package main

import (
 "context"
 "os"
 "path/filepath"
 "testing"

 "github.com/Routilet/companion/internal/localrun"
 "github.com/Routilet/companion/internal/ojadapter"
)

type fakeImportAccount struct {
 *ojadapter.Accounts
 result ojadapter.ImportResult
 id string
}
func (f *fakeImportAccount) Status() ojadapter.AccountView {return ojadapter.AccountView{AccountState:ojadapter.AccountState{State:"signed_in",ID:f.id},Available:true}}
func (f *fakeImportAccount) Import(context.Context,string)(ojadapter.ImportResult,error){return f.result,nil}
func importProblem(id,label string)ojadapter.Problem {
 target:=ojadapter.Target{Kind:"contest",ContestID:"42",ProblemID:id}
 return ojadapter.Problem{Target:target,Title:"题目 "+label,Label:label,SourceURL:target.URL(),Samples:[]ojadapter.Sample{{Input:"1\n",Expected:"2\n"}}}
}
func importDesktop(t *testing.T,ps ...ojadapter.Problem)(*Desktop,*fakeImportAccount){
 t.Helper();root,e:=localrun.OpenDirectory(t.TempDir());if e!=nil{t.Fatal(e)}
 a:=&fakeImportAccount{id:"7",result:ojadapter.ImportResult{AccountID:"7",Title:"作业",Problems:ps}}
 return &Desktop{root:root,accounts:a},a
}
func TestOJImportDesktop(t *testing.T){
 d,_:=importDesktop(t,importProblem("1","A"),importProblem("2","B"))
 a,b:=filepath.Join(d.root,"A.cpp"),filepath.Join(d.root,"B.cpp")
 original:=[]byte("int main(){return 0;}\n")
 if e:=os.WriteFile(b,original,0600);e!=nil{t.Fatal(e)}
 p,e:=d.OJPreviewImport("fixture",".cpp",false);if e!=nil{t.Fatal(e)}
 if len(p.Plan.Rows)!=2||p.Plan.Rows[0].Action!="create"||p.Plan.Rows[1].Action!="skip"{t.Fatal(p.Plan)}
 if _,e=os.Stat(a);!os.IsNotExist(e){t.Fatal("preview wrote source",e)}
 r,e:=d.OJApplyImport(p.Token);if e!=nil||len(r.Completed)!=1||len(r.Skipped)!=1||len(r.Failed)!=0{t.Fatal(r,e)}
 cs,e:=localrun.LoadCases(a);if e!=nil||len(cs.Cases)!=1||cs.Cases[0].Expected==nil||*cs.Cases[0].Expected!="2\n"{t.Fatal(cs,e)}
 bind,e:=d.OJBinding();if e!=nil||bind==nil||bind.Target.ProblemID!="1"{t.Fatal(bind,e)}
 if raw,e:=os.ReadFile(b);e!=nil||string(raw)!=string(original){t.Fatal("conflicting source overwritten",e)}
 if _,e=d.OJApplyImport(p.Token);e==nil{t.Fatal("replayed plan")}
 if _,e=d.Rename("renamed.cpp");e!=nil{t.Fatal(e)}
 next:=filepath.Join(d.root,"renamed.cpp");bind,e=d.OJBinding();if e!=nil||bind==nil{t.Fatal(bind,e)}
 if old,e:=ojadapter.LoadBinding(a);e!=nil||old!=nil{t.Fatal(old,e)}
 if e=d.OJUnbind(next);e!=nil{t.Fatal(e)}
 if c,e:=localrun.LoadCases(next);e!=nil||len(c.Cases)!=1{t.Fatal(c,e)}
 if _,e=os.Stat(next);e!=nil{t.Fatal(e)}
}
func TestOJAttachExistingAndAccountChange(t *testing.T){
 p:=importProblem("5","A");p.Target=ojadapter.Target{Kind:"problemset",ProblemID:"5"};p.SourceURL=p.Target.URL();p.Label="#5"
 d,account:=importDesktop(t,p)
 s,e:=localrun.CreateSourceWithKit(d.root,d.root,"existing.cpp","");if e!=nil{t.Fatal(e)};d.selected=s
 custom:="custom"
 cs,e:=localrun.SaveCases(s.Path,[]localrun.Case{{Input:"mine",Expected:&custom}},"missing");if e!=nil{t.Fatal(e)}
 preview,e:=d.OJPreviewImport("fixture",".cpp",true);if e!=nil||preview.Plan.Rows[0].Action!="attach"||preview.Plan.Rows[0].Samples!=0{t.Fatal(preview,e)}
 account.id="8";if _,e=d.OJApplyImport(preview.Token);e==nil{t.Fatal("account changed")}
 account.id="7";preview,e=d.OJPreviewImport("fixture",".cpp",true);if e!=nil{t.Fatal(e)}
 res,e:=d.OJApplyImport(preview.Token);if e!=nil||len(res.Completed)!=1||len(res.Failed)!=0{t.Fatal(res,e)}
 after,e:=localrun.LoadCases(s.Path);if e!=nil||after.Revision!=cs.Revision{t.Fatal(after,e)}
}

func TestIntegratedCreateLocationAndBindingGuard(t *testing.T){
 d,_:=importDesktop(t)
 oldRoot:=d.root
 other,e:=localrun.OpenDirectory(t.TempDir());if e!=nil{t.Fatal(e)}
 old:=filepath.Join(other,"old.cpp")
 if e=ojadapter.SaveBinding(old,importProblem("5","A").Binding());e!=nil{t.Fatal(e)}
 if _,e=d.Create(other,"old.cpp");e==nil{t.Fatal("old binding was reused")}
 if d.root!=oldRoot{t.Fatal("failed create changed root")}
 if _,e=os.Stat(old);!os.IsNotExist(e){t.Fatal("failed create wrote source",e)}
 nav,e:=d.Create(other,"fresh.cpp");if e!=nil{t.Fatal(e)}
 if nav.Root!=other||nav.Source.Path!=filepath.Join(other,"fresh.cpp"){t.Fatal(nav)}
}

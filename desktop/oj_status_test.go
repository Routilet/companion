package main
import("context";"testing";"github.com/Routilet/companion/internal/ojadapter")
type fakeJudgeAccount struct{*fakeImportAccount;calls int}
func(f *fakeJudgeAccount)Judge(_ context.Context,t ojadapter.Target,id string)(ojadapter.JudgeStatus,error){
 f.calls++;if id==""{return ojadapter.JudgeStatus{AccountID:f.id,Target:t,Accepted:true,Latest:&ojadapter.JudgeResult{ID:"124",Result:"WA",Language:"C++17"}},nil}
 return ojadapter.JudgeStatus{AccountID:f.id,Target:t,Accepted:true,Latest:&ojadapter.JudgeResult{ID:id,Result:"AC",Language:"C++17"}},nil
}
func TestOJJudgeDesktop(t *testing.T){
 p:=importProblem("5","A");d,a:=importDesktop(t,p);f:=&fakeJudgeAccount{fakeImportAccount:a};d.accounts=f
 plan,e:=d.OJPreviewImport("fixture",".cpp",false);if e!=nil{t.Fatal(e)};if _,e=d.OJApplyImport(plan.Token);e!=nil{t.Fatal(e)}
 path:=d.selected.Path
 if v:=d.OJCachedJudges([]string{path});len(v)!=0||f.calls!=0{t.Fatal("cache initiated network",v)}
 d.rememberReceipt(ojadapter.Receipt{ID:"123",Target:p.Target,AccountID:"7",Language:"C++17",Result:"WT"})
 if n,e:=d.OJRefreshJudges([]string{path,path});e!=nil||n!=1||f.calls!=1{t.Fatal(n,e,f.calls)}
 if v,e:=d.OJPollReceipt("123");e!=nil||v.Result!="AC"{t.Fatal(v,e)}
 v:=d.OJCachedJudges([]string{path})[path].Status;if !v.Accepted||v.Latest.ID!="124"||v.Latest.Result!="WA"{t.Fatal(v)}
 a.id="8";if len(d.OJCachedJudges([]string{path}))!=0{t.Fatal("other account cache")};if _,e=d.OJPollReceipt("123");e==nil{t.Fatal("polled other account")}
 a.id="7";if e=d.OJUnbind(path);e!=nil{t.Fatal(e)};if len(d.OJCachedJudges([]string{path}))!=0{t.Fatal("unbound cache")}
}

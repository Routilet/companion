package main

import (
 "context"
 "os"
 "testing"
 "github.com/Routilet/companion/internal/localrun"
 "github.com/Routilet/companion/internal/ojadapter"
)

type fakeSubmitAccount struct {
 *fakeImportAccount
 sent int
 code string
}
func(f *fakeSubmitAccount) PrepareOnline(_ context.Context,t ojadapter.Target,code string)(ojadapter.SubmitPlan,error){
 return ojadapter.SubmitPlan{Token:"0123456789abcdef0123456789abcdef",Target:t,AccountID:f.id,Languages:[]string{"C++17"},SourceHash:ojadapter.SourceHash(code)},nil
}
func(f *fakeSubmitAccount) SubmitOnline(_ context.Context,p ojadapter.SubmitPlan,lang,code string)(ojadapter.Receipt,error){
 f.sent++;f.code=code
 return ojadapter.Receipt{ID:"123",Target:p.Target,AccountID:p.AccountID,Language:lang,Result:"WT",SourceHash:p.SourceHash},nil
}
func TestOJSubmitDesktop(t *testing.T){
 p:=importProblem("5","A")
 d,a:=importDesktop(t,p)
 f:=&fakeSubmitAccount{fakeImportAccount:a};d.accounts=f
 s,e:=localrun.CreateSourceWithKit(d.root,d.root,"A.cpp","");if e!=nil{t.Fatal(e)}
 d.selected=s
 if d.OJSubmissionAvailable(){t.Fatal("unbound available")}
 if e=ojadapter.SaveBinding(s.Path,p.Binding());e!=nil{t.Fatal(e)}
 if !d.OJSubmissionAvailable(){t.Fatal("bound unavailable")}
 preview,e:=d.Prepare("");if e!=nil{t.Fatal(e)}
 plan,e:=d.OJPrepareSubmission("",preview.Digest);if e!=nil{t.Fatal(e)}
 if e=os.WriteFile(s.Path,[]byte("int main(){return 3;}\n"),0600);e!=nil{t.Fatal(e)}
 if _,e=d.OJSendSubmission("",preview.Digest,plan.Plan.Token,"C++17");e==nil||f.sent!=0{t.Fatal("changed source sent",e,f.sent)}
 preview,e=d.Prepare("");if e!=nil{t.Fatal(e)}
 plan,e=d.OJPrepareSubmission("",preview.Digest);if e!=nil{t.Fatal(e)}
 a.id="8";if _,e=d.OJSendSubmission("",preview.Digest,plan.Plan.Token,"C++17");e==nil||f.sent!=0{t.Fatal("changed account sent",e)}
 a.id="7";plan,e=d.OJPrepareSubmission("",preview.Digest);if e!=nil{t.Fatal(e)}
 r,e:=d.OJSendSubmission("",preview.Digest,plan.Plan.Token,"C++17");if e!=nil||r.ID!="123"||f.sent!=1||f.code!=preview.Text{t.Fatal(r,e,f.sent)}
 if _,e=d.OJSendSubmission("",preview.Digest,plan.Plan.Token,"C++17");e==nil||f.sent!=1{t.Fatal("replayed",e)}
}

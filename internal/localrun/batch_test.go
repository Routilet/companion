package localrun

import (
 "context"
 "os"
 "path/filepath"
 "strings"
 "testing"
 "time"
)
func str(s string)*string{return &s}
func TestCasesPersistence(t *testing.T){
 p:=source(t,"int main(void){return 0;}")
 cs,e:=LoadCases(p);if e!=nil||cs.Revision!="missing"||len(cs.Cases)!=0{t.Fatalf("%+v %v",cs,e)}
 if _,e=os.Stat(filepath.Join(filepath.Dir(p),".companion"));!os.IsNotExist(e){t.Fatal("read created metadata")}
 cs,e=SaveCases(p,[]Case{{"a",str("")},{"",nil}},cs.Revision);if e!=nil{t.Fatal(e)}
 got,e:=LoadCases(p);if e!=nil||got.Revision!=cs.Revision||got.Cases[0].Expected==nil||*got.Cases[0].Expected!=""||got.Cases[1].Expected!=nil{t.Fatalf("%+v %v",got,e)}
 if _,e=SaveCases(p,nil,"missing");e==nil{t.Fatal("stale write accepted")}
 next,e:=SaveCases(p,[]Case{},got.Revision);if e!=nil||next.Revision==got.Revision{t.Fatalf("%+v %v",next,e)}
 path,_:=casePath(p);bad:=[]byte("{invalid");if e=os.WriteFile(path,bad,0600);e!=nil{t.Fatal(e)}
 if _,e=LoadCases(p);e==nil{t.Fatal("corrupt read accepted")}
 if _,e=SaveCases(p,nil,next.Revision);e==nil{t.Fatal("corrupt overwrite accepted")}
 b,_:=os.ReadFile(path);if string(b)!=string(bad){t.Fatal("corrupt data overwritten")}
}
func TestCompare(t *testing.T){
 for _,pair:=range [][2]string{{"a \r\n\n","a\n"},{""," \t\n"},{"a\n\nb","a\n\nb"}}{if Compare(pair[0],pair[1])!=nil{t.Fatal(pair)}}
 for _,pair:=range [][2]string{{" a","a"},{"a b","ab"},{"a\nb","a\n\nb"},{"","x"}}{if Compare(pair[0],pair[1])==nil{t.Fatal(pair)}}
 d:=Compare("你好\nA","你好\nB");if d.Line!=2||d.Column!=1{t.Fatalf("%+v",d)}
}
func TestBatchAndStale(t *testing.T){
 requireCompiler(t)
 p:=source(t,"#include <stdio.h>\nint main(void){int c;while((c=getchar())!=EOF)putchar(c);return 0;}\n")
 cs,e:=SaveCases(p,[]Case{{"1",str("1\n")},{"2",str("3")},{"",str("")},{"4",nil}},"missing");if e!=nil{t.Fatal(e)}
 r,e:=RunBatch(context.Background(),p,"",cs.Revision,-1)
 if e!=nil||r.State!="completed"||r.Stale||len(r.Cases)!=4||r.RunID==""{t.Fatalf("%+v %v",r,e)}
 for i,w:=range []string{"passed","mismatch","passed","ran"}{if r.Cases[i].State!=w{t.Fatalf("%d: %+v",i,r.Cases[i])}}
 if !Current(r,""){t.Fatal("fresh batch marked stale")}
 if _,e=SaveCases(p,nil,cs.Revision);e!=nil{t.Fatal(e)}
 if Current(r,""){t.Fatal("case change ignored")}
 if _,e=RunBatch(context.Background(),p,"",cs.Revision,0);e==nil{t.Fatal("old revision accepted")}
}
func TestCancel(t *testing.T){
 requireCompiler(t)
 p:=source(t,"int main(void){for(;;){}}")
 ctx,cancel:=context.WithTimeout(context.Background(),500*time.Millisecond);defer cancel()
 r,e:=RunBatch(ctx,p,"","missing",-1)
 if e!=nil||r.State!="cancelled"{t.Fatalf("%+v %v",r,e)}
}
func TestDiagnostics(t *testing.T){
 d:=diagnostics("C:\\temp\\source.c:7:4: error: bad\n detail\n","source.c","A.c")
 if len(d)!=1||d[0].Line!=7||d[0].Column!=4||d[0].Severity!="error"||!strings.Contains(d[0].Raw,"detail"){t.Fatalf("%+v",d)}
}
func TestSourceChange(t *testing.T){
 requireCompiler(t)
 p:=source(t,"int main(void){return 0;}")
 r,e:=RunBatch(context.Background(),p,"","missing",-1);if e!=nil{t.Fatal(e)}
 if e=os.WriteFile(p,[]byte("int main(void){return 1;}"),0600);e!=nil{t.Fatal(e)}
 if Current(r,""){t.Fatal("source change ignored")}
}

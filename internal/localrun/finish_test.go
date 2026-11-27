package localrun

import (
 "context"
 "os"
 "path/filepath"
 "testing"
)
func TestSubmission(t *testing.T){
 requireCompiler(t)
 p:=source(t,"int main(void){return 0;}\n")
 cs,e:=SaveCases(p,[]Case{{"",str("")}},"missing");if e!=nil{t.Fatal(e)}
 r,e:=RunBatch(context.Background(),p,"",cs.Revision,-1);if e!=nil{t.Fatal(e)}
 snap,e:=PrepareSubmission(p,"",&r);if e!=nil||!snap.Verified{t.Fatalf("%+v %v",snap,e)}
 if _,e=CheckSubmission(snap,"");e!=nil{t.Fatal(e)}
 if e=os.WriteFile(p,[]byte("int main(void){return 1;}\n"),0600);e!=nil{t.Fatal(e)}
 if _,e=CheckSubmission(snap,"");e==nil{t.Fatal("copied changed source")}
}
func TestSettingsRoundTrip(t *testing.T){
 p:=filepath.Join(t.TempDir(),"Companion","settings.json")
 s,e:=LoadSettings(p);if e!=nil||s!=(Settings{}){t.Fatal(s,e)}
 want:=Settings{Root:"中文",C:"gcc",CPP:"g++",Editor:"code"}
 if e=SaveSettings(p,want);e!=nil{t.Fatal(e)}
 s,e=LoadSettings(p);if e!=nil||s!=want{t.Fatal(s,e)}
}
func TestDiffParts(t *testing.T){
 d:=Compare("aX中Yz","aQ中Rz");if d==nil||d.Coarse{t.Fatal(d)}
 var expected,actual string;var changes int
 for _,p:=range d.ExpectedParts{expected+=p.Text;if p.Changed{changes++}}
 for _,p:=range d.ActualParts{actual+=p.Text}
 if expected!="aX中Yz"||actual!="aQ中Rz"||changes!=2{t.Fatalf("%+v",d)}
}
func TestProgress(t *testing.T){
 requireCompiler(t)
 p:=source(t,"int main(void){return 0;}")
 cs,e:=SaveCases(p,[]Case{{"",str("")},{"",str("")}},"missing");if e!=nil{t.Fatal(e)}
 var events []Progress
 r,e:=RunBatchProgress(context.Background(),p,"",cs.Revision,-1,func(p Progress){events=append(events,p)})
 if e!=nil||len(events)!=4||events[0].Phase!="compiling"||events[3].Completed!=2||events[3].RunID!=r.RunID{t.Fatalf("%+v %v",events,e)}
}

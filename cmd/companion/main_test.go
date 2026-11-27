package main

import (
 "bytes"
 "os"
 "path/filepath"
 "strings"
 "testing"
 "github.com/Routilet/companion/internal/localrun"
)
func TestSourceCommand(t *testing.T){
 path:=filepath.Join(t.TempDir(),"中文 A.c")
 raw:="#include <stdio.h>\nint main(void){return 0;}\n"
 if e:=os.WriteFile(path,[]byte(raw),0600);e!=nil{t.Fatal(e)}
 var out,errOut bytes.Buffer
 if code:=run([]string{"source",path},&out,&errOut);code!=0||out.String()!=raw||errOut.Len()!=0{t.Fatalf("%d %q %q",code,out.String(),errOut.String())}
 out.Reset();errOut.Reset()
 if code:=run([]string{"source",path+".missing"},&out,&errOut);code!=2||out.Len()!=0||errOut.Len()==0{t.Fatal(code,out.String(),errOut.String())}
}
func TestCLIExitCodes(t *testing.T){
 path:=filepath.Join(t.TempDir(),"A.c")
 if e:=os.WriteFile(path,[]byte("#include <stdio.h>\nint main(void){puts(\"ok\");return 0;}\n"),0600);e!=nil{t.Fatal(e)}
 cs,e:=localrun.LoadCases(path);if e!=nil{t.Fatal(e)}
 yes,no:="ok\n","bad"
 if _,e=localrun.SaveCases(path,[]localrun.Case{{Input:"",Expected:&yes},{Input:"",Expected:&no}},cs.Revision);e!=nil{t.Fatal(e)}
 var out,errOut bytes.Buffer
 if code:=run([]string{"test","-case","1",path},&out,&errOut);code!=0||!strings.Contains(out.String(),"passed"){t.Fatal(code,out.String(),errOut.String())}
 out.Reset();errOut.Reset()
 if code:=run([]string{"test",path},&out,&errOut);code!=1||!strings.Contains(out.String(),"mismatch"){t.Fatal(code,out.String(),errOut.String())}
}

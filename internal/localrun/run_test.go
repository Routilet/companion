package localrun

import (
 "context"
 "os"
 "path/filepath"
 "strings"
 "testing"
)
func source(t *testing.T,code string)string{
 t.Helper();d:=filepath.Join(t.TempDir(),"中文 空格")
 if e:=os.Mkdir(d,0700);e!=nil{t.Fatal(e)}
 p:=filepath.Join(d,"A.c");if e:=os.WriteFile(p,[]byte(code),0600);e!=nil{t.Fatal(e)};return p
}
func requireCompiler(t *testing.T){
 t.Helper()
 if _,e:=compilerPath("");e!=nil{t.Fatal(e)}
}
func TestOpenNoWrites(t *testing.T){
 p:=source(t,"int main(void){return 0;}")
 s,e:=Open(p);if e!=nil||s.Directory!=filepath.Dir(p){t.Fatalf("%+v %v",s,e)}
 entries,e:=os.ReadDir(s.Directory);if e!=nil||len(entries)!=1{t.Fatalf("%v %v",entries,e)}
}
func TestRunExactInput(t *testing.T){
 requireCompiler(t)
 code:="#include <stdio.h>\nint main(void){int c;while((c=getchar())!=EOF)putchar(c);return 0;}\n"
 p:=source(t,code)
 r,e:=Run(context.Background(),p,"hello 42","")
 if e!=nil||r.State!="ran"||r.Execution==nil||r.Execution.Stdout!="hello 42"||r.Stale{t.Fatalf("%+v %v",r,e)}
 b,_:=os.ReadFile(p);if string(b)!=code{t.Fatal("source changed")}
 entries,_:=os.ReadDir(filepath.Dir(p));if len(entries)!=1{t.Fatalf("source directory polluted: %v",entries)}
}
func TestCompileAndRuntimeFailure(t *testing.T){
 requireCompiler(t)
 r,e:=Run(context.Background(),source(t,"not valid C"),"","")
 if e!=nil||r.State!="compilation_failed"||r.Execution!=nil||r.Compilation.Stderr==""{t.Fatalf("%+v %v",r,e)}
 r,e=Run(context.Background(),source(t,"int main(void){return 3;}"),"","")
 if e!=nil||r.State!="runtime_failed"||r.Execution==nil||r.Execution.ExitCode!=3{t.Fatalf("%+v %v",r,e)}
}
func TestMissingCompilerAndLocalHeader(t *testing.T){
 p:=source(t,"int main(void){return 0;}")
 if _,e:=Run(context.Background(),p,"",filepath.Join(t.TempDir(),"missing-cc"));e==nil||!strings.Contains(e.Error(),"编译器"){t.Fatalf("%v",e)}
 if _,e:=Run(context.Background(),source(t,"#include \"exam.h\"\n"),"","");e==nil{t.Fatal("local include accepted")}
}
func TestTimeout(t *testing.T){
 requireCompiler(t)
 r,e:=Run(context.Background(),source(t,"int main(void){for(;;){} }"),"","")
 if e!=nil||r.State!="timeout"{t.Fatalf("%+v %v",r,e)}
}
func TestOutputLimit(t *testing.T){
 requireCompiler(t)
 r,e:=Run(context.Background(),source(t,"#include <stdio.h>\nint main(void){for(;;)puts(\"0123456789abcdefghijklmnopqrstuvwxyz\");}"),"","")
 if e!=nil||r.State!="output_limit"||r.Execution==nil||len(r.Execution.Stdout)+len(r.Execution.Stderr)>maxOutput{t.Fatalf("state=%s err=%v",r.State,e)}
}

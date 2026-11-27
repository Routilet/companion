package localrun

import (
 "encoding/json"
 "context"
 "runtime"
 "time"
 "os"
 "path/filepath"
 "strings"
 "testing"
)
func TestObserverSession(t *testing.T){
 requireCompiler(t)
 s:=kitSource(t,kitFixture(t,"0.6.10"),"int main(void){\n  /* comment */ int x=read_int();\n  printf(\"%d\\n\",x);\n  return 0;\n}\n")
 cs,e:=SaveCases(s.Path,[]Case{{"7\n",str("")},{"8\n",nil}},"missing");if e!=nil{t.Fatal(e)}
 for i:=range cs.Cases{
  o,e:=PrepareObservation(s.Path,"",cs.Revision,i,ObserverInfo{Backend:"lldb"});if e!=nil{t.Fatal(e)}
  raw,e:=os.ReadFile(o.SessionPath);if e!=nil{t.Fatal(e)}
  var config observerSession;if e=json.Unmarshal(raw,&config);e!=nil{t.Fatal(e)}
  if config.SchemaVersion!=1||config.WorkDir!=s.Directory||config.Comparison!="trim-line-end"||config.SourceMap["companion-user.c"]!="user.c"||len(config.CompilerArgs)!=1||config.CompilerArgs[0]!="-lm"{t.Fatalf("%+v",config)}
  frozen,_:=os.ReadFile(filepath.Join(o.Directory,"user.c"));original,_:=os.ReadFile(s.Path)
  if string(frozen)!=string(original){t.Fatal("user snapshot differs")}
  compiled,_:=os.ReadFile(filepath.Join(o.Directory,"source.c"))
  if !strings.Contains(string(compiled),"#line")||!strings.Contains(string(compiled),"/* comment */ int x=read_int();")||strings.Contains(string(compiled),"expanded.h"){t.Fatal(string(compiled))}
  if (config.ExpectedStdoutFile==nil)!=(i==1){t.Fatal("null and empty confused")}
  if i==0{b,e:=os.ReadFile(filepath.Join(o.Directory,*config.ExpectedStdoutFile));if e!=nil||len(b)!=0{t.Fatal(e)}}
  if e=o.Cleanup();e!=nil{t.Fatal(e)}
  if _,e=os.Stat(o.Directory);!os.IsNotExist(e){t.Fatal("session leaked")}
 }
 if _,e=PrepareObservation(s.Path,"","wrong",0,ObserverInfo{});e==nil{t.Fatal("stale cases accepted")}
}
func TestObserverPlain(t *testing.T){
 requireCompiler(t)
 p:=source(t,"int main(void){return 0;}\n")
 o,e:=PrepareObservation(p,"","missing",0,ObserverInfo{Backend:"gdb"});if e!=nil{t.Fatal(e)}
 defer o.Cleanup()
 b,_:=os.ReadFile(filepath.Join(o.Directory,"source.c"));original,_:=os.ReadFile(p)
 if string(b)!=string(original){t.Fatal("plain source changed")}
}
func TestObserverManifest(t *testing.T){
 root:=t.TempDir();b:=[]byte("fixture")
 if e:=os.WriteFile(filepath.Join(root,"observer"),b,0600);e!=nil{t.Fatal(e)}
 m:=observerManifest{1,"0.2.0",[]int{1},map[string]observerTarget{"test":{"observer",digest(b),"stable"}}}
 save:=func(){raw,_:=json.Marshal(m);if e:=os.WriteFile(filepath.Join(root,"manifest.json"),raw,0600);e!=nil{t.Fatal(e)}}
 save();if _,_,e:=resolveObserver(root,"test");e!=nil{t.Fatal(e)}
 m.Platforms["test"]=observerTarget{"../observer",digest(b),"stable"};save()
 if _,_,e:=resolveObserver(root,"test");e==nil{t.Fatal("path traversal accepted")}
 m.Platforms["test"]=observerTarget{"observer",strings.Repeat("0",64),"stable"};save()
 if _,_,e:=resolveObserver(root,"test");e==nil{t.Fatal("checksum mismatch accepted")}
}
func TestReleasedObserverCapabilities(t *testing.T){
 root:=os.Getenv("COMPANION_TEST_OBSERVER_PACKAGE");if root==""{t.Skip("release package not provided")}
 info,e:=ProbeObserver(root);if e!=nil{t.Fatal(e)}
 t.Logf("Observer %s / %s capabilities verified",info.Version,info.Backend)
}

func TestObserverRealKitCompile(t *testing.T){
 pkg:=os.Getenv("COMPANION_TEST_KIT_PACKAGE");if pkg==""{t.Skip("real package fixture not provided")}
 requireCompiler(t)
 s:=kitSource(t,pkg,"int main(void){\n  /* keep the exact original line */ int x=read_int();\n  printf(\"%d\\n\",x);\n  return 0;\n}\n")
 cs,e:=SaveCases(s.Path,[]Case{{"42\n",str("42\n")}},"missing");if e!=nil{t.Fatal(e)}
 o,e:=PrepareObservation(s.Path,"",cs.Revision,0,ObserverInfo{Backend:"lldb"});if e!=nil{t.Fatal(e)}
 defer o.Cleanup()
 cc,e:=compilerPath("");if e!=nil{t.Fatal(e)}
 program:=filepath.Join(o.Directory,"program");if runtime.GOOS=="windows"{program+=".exe"}
 r:=process(context.Background(),cc,[]string{"-std=c99","-g","-O0",artifactArg(s.Directory,filepath.Join(o.Directory,"source.c")),"-lm","-o",artifactArg(s.Directory,program)},s.Directory,"",childEnv(cc),30*time.Second)
 if r.State!="ok"{t.Fatalf("observation source compile: %+v",r)}
 r=process(context.Background(),program,nil,s.Directory,"42\n",childEnv(cc),2*time.Second)
 if r.State!="ok"||normalize(r.Stdout)!="42"{t.Fatalf("observation execution: %+v",r)}
}

func TestObserverCapabilityBackendNames(t *testing.T){
 for _,v:=range []struct{platform,arch,selector,reported string}{{"windows","amd64","gdb","gdb-mi"},{"darwin","arm64","lldb","lldb-dap"}}{
  c:=observerCapabilities{Version:"0.2.0",Platform:v.platform,Arch:v.arch,Backends:[]string{v.reported},SessionSchemaVersions:[]int{1},SessionFeatures:[]string{"backend","comparison"}}
  if e:=validateObserverCapabilities(c,"0.2.0",v.platform,v.arch,v.selector);e!=nil{t.Fatal(e)}
  c.Backends=[]string{v.selector}
  if e:=validateObserverCapabilities(c,"0.2.0",v.platform,v.arch,v.selector);e==nil{t.Fatal("session selector confused with backend capability ID")}
 }
}

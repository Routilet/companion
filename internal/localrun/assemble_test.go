package localrun

import (
 "bytes"
 "context"
 "os"
 "path/filepath"
 "strings"
 "testing"
)
func kitSource(t *testing.T,pkg,body string)Source{
 t.Helper();root,e:=OpenDirectory(t.TempDir());if e!=nil{t.Fatal(e)}
 s,e:=CreateSourceWithKit(root,root,"A.c",pkg);if e!=nil{t.Fatal(e)}
 b,e:=os.ReadFile(s.Path);if e!=nil{t.Fatal(e)}
 line:=strings.SplitN(string(b),"\n",2)[0]
 if e=os.WriteFile(s.Path,[]byte(line+"\n"+body),0600);e!=nil{t.Fatal(e)}
 return s
}
func TestKitAssemblyAndSelection(t *testing.T){
 pkg:=kitFixture(t,"0.6.10")
 s:=kitSource(t,pkg,"int main(void){ /* read_int() */ puts(\"read_int // literal\"); return 0; }\n")
 _,p,e:=PreparePath(s.Path);if e!=nil{t.Fatal(e)}
 if len(p.Components)!=0||strings.Contains(string(p.Bytes),"static inline")||!strings.Contains(string(p.Bytes),"\"read_int // literal\"")||strings.Contains(string(p.Bytes),"/*"){t.Fatal(string(p.Bytes),p.Components)}
 s=kitSource(t,pkg,"int main(void){int/**/x=read_int(); printf(\"%d\\n\",x);return 0;}\n")
 _,p,e=PreparePath(s.Path);if e!=nil{t.Fatal(e)}
 if strings.Join(p.Components,",")!="helper,read_int"||strings.Contains(string(p.Bytes),"expanded.h")||strings.Contains(string(p.Bytes),"intx"){t.Fatal(string(p.Bytes),p.Components)}
 if len(sourceLines(p.Bytes))!=len(p.Origins){t.Fatal("line origins do not match generated file")}
}
func TestKitRestrictions(t *testing.T){
 pkg:=kitFixture(t,"0.6.10")
 for _,body:=range []string{
  "#define X 1\nint main(void){return X;}\n",
  "#include \"other.h\"\nint main(void){return 0;}\n",
  "int read_int(void){return 0;} int main(void){return read_int();}\n",
  "int helper(void){return 0;} int main(void){return read_int();}\n",
  "int main(void){return read_\\\nint();}\n",
 }{
  s:=kitSource(t,pkg,body);if _,_,e:=PreparePath(s.Path);e==nil{t.Fatalf("accepted unsupported source %q",body)}
 }
 s:=kitSource(t,pkg,"int main(void){return 0;}\n")
 raw,_:=os.ReadFile(s.Path)
 if e:=os.WriteFile(s.Path,append([]byte("#if 1\n"),append(raw,[]byte("#endif\n")...)...),0600);e!=nil{t.Fatal(e)}
 if _,_,e:=PreparePath(s.Path);e==nil{t.Fatal("conditional include accepted")}
}
func TestKitRunCopyAndStale(t *testing.T){
 requireCompiler(t)
 s:=kitSource(t,kitFixture(t,"0.6.10"),"int main(void){printf(\"%d\\n\",read_int());return 0;}\n")
 cs,e:=SaveCases(s.Path,[]Case{{"17\n",str("17\n")}},"missing");if e!=nil{t.Fatal(e)}
 r,e:=RunBatch(context.Background(),s.Path,"",cs.Revision,-1)
 if e!=nil||r.State!="completed"||r.Stale||len(r.Cases)!=1||r.Cases[0].State!="passed"{t.Fatalf("%+v %v",r,e)}
 p,e:=PrepareSubmission(s.Path,"",&r);if e!=nil||!p.Verified||p.Digest!=r.PreparedDigest||digest([]byte(p.Text))!=r.PreparedDigest||p.ResourceDigest!=r.ResourceDigest{t.Fatalf("%+v %v",p,e)}
 header:=filepath.Join(s.Directory,".companion","resources","0.6.10","expanded.h")
 if e=os.WriteFile(header,[]byte("tampered"),0600);e!=nil{t.Fatal(e)}
 if Current(r,""){t.Fatal("tampered resource not stale")}
 if _,e=CheckSubmission(p,"");e==nil{t.Fatal("tampered resource copied")}
}
func TestKitDiagnostics(t *testing.T){
 requireCompiler(t)
 s:=kitSource(t,kitFixture(t,"0.6.10"),"int main(void){\n  int x = read_int();\n  return unknown;\n}\n")
 r,e:=RunBatch(context.Background(),s.Path,"","missing",-1)
 if e!=nil||r.State!="compilation_failed"{t.Fatalf("%+v %v",r,e)}
 for _,d:=range r.Diagnostics{if d.File==s.Path&&d.Line==4&&strings.Contains(d.Message,"unknown"){return}}
 t.Fatalf("diagnostic not mapped to source: %+v",r.Diagnostics)
}
func TestActualKitMinimalAssembly(t *testing.T){
 pkg:=os.Getenv("COMPANION_TEST_KIT_PACKAGE");if pkg==""{t.Skip("real package fixture not provided")}
 requireCompiler(t)
 s:=kitSource(t,pkg,"int main(void){int n=read_int();printf(\"%d\\n\",n);return 0;}\n")
 _,p,e:=PreparePath(s.Path);if e!=nil{t.Fatal(e)}
 k,e:=LoadKitPackage(pkg);if e!=nil{t.Fatal(e)}
 if len(p.Components)<2||len(p.Components)>=len(k.Order)||bytes.Contains(p.Bytes,[]byte("expanded.h")){t.Fatal(p.Components)}
 cases,e:=SaveCases(s.Path,[]Case{{"42\n",str("42\n")}},"missing");if e!=nil{t.Fatal(e)}
 r,e:=RunBatch(context.Background(),s.Path,"",cases.Revision,-1)
 if e!=nil||r.State!="completed"||len(r.Cases)!=1||r.Cases[0].State!="passed"{t.Fatalf("%+v %v",r,e)}
 submit,e:=PrepareSubmission(s.Path,"",&r);if e!=nil||!submit.Verified||!bytes.Equal([]byte(submit.Text),p.Bytes){t.Fatal("submission differs",e)}
 t.Logf("actual kit %s: selected %d / %d declarations, prepared %d bytes",k.Version,len(p.Components),len(k.Order),len(p.Bytes))
}
func TestPlainCOriginalBytes(t *testing.T){
 s:=source(t,"// #include \"fake.h\"\n#define X 2\nint main(void){return X;}\n")
 raw,_:=os.ReadFile(s);_,p,e:=PreparePath(s)
 if e!=nil||!bytes.Equal(raw,p.Bytes)||p.ResourceDigest!=""{t.Fatal(e)}
}

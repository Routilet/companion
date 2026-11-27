package localrun

import (
 "bytes"
 "context"
 "encoding/json"
 "os"
 "path/filepath"
 "runtime"
 "strings"
 "testing"
 "time"
)
func kitFixture(t *testing.T,version string)string{
 t.Helper();root:=t.TempDir();dir:=filepath.Join(root,"preset")
 if e:=os.Mkdir(dir,0700);e!=nil{t.Fatal(e)}
 manifest:=KitManifest{SchemaVersion:2,ID:"exam-io-c99",Language:"c",LanguageID:"c99",Standard:"c99",HeaderFile:"exam.h",SystemHeaders:[]string{"stdio.h"},Components:[]KitComponent{
  {"read_int",[]string{"read_int"},[]string{"helper"},"static inline int read_int(void){return helper();}\n"},
  {"helper",[]string{},[]string{},"static inline int helper(void){int x; scanf(\"%d\",&x);return x;}\n"},
 }}
 m,_:=json.Marshal(manifest)
 for path,data:=range map[string][]byte{
  "package.json":[]byte("{\"name\":\"@routilet/c-exam-kit\",\"version\":\""+version+"\"}"),
  "LICENSE":[]byte("fixture license"),
  "preset/CC-BY-4.0.txt":[]byte("fixture documentation license"),
  "preset/manifest.json":m,
  "preset/reference.json":[]byte("{\"schemaVersion\":1,\"version\":\""+version+"\",\"categories\":[{}],\"sections\":[{}]}"),
 }{if e:=os.WriteFile(filepath.Join(root,filepath.FromSlash(path)),data,0600);e!=nil{t.Fatal(e)}}
 return root
}
func TestKitVersionsAndIntegrity(t *testing.T){
 root,e:=OpenDirectory(t.TempDir());if e!=nil{t.Fatal(e)}
 sub:=filepath.Join(root,"作业");if e=os.Mkdir(sub,0700);e!=nil{t.Fatal(e)}
 pkg:=kitFixture(t,"0.6.10");k,e:=LoadKitPackage(pkg);if e!=nil{t.Fatal(e)}
 if strings.Join(k.Order,",")!="helper,read_int"{t.Fatal(k.Order)}
 a,e:=CreateSourceWithKit(root,sub,"A.c",pkg);if e!=nil{t.Fatal(e)}
 raw,e:=os.ReadFile(a.Path);if e!=nil||!strings.HasPrefix(string(raw),"#include \"../.companion/resources/0.6.10/expanded.h\"\n"){t.Fatal(string(raw),e)}
 old:=filepath.Join(root,".companion","resources","0.6.10")
 saved,e:=LoadKitSnapshot(old);if e!=nil||saved.Fingerprint!=k.Fingerprint{t.Fatal(e)}
 if _,e=SaveKitSnapshot(root,k);e!=nil{t.Fatal(e)}
 newer:=kitFixture(t,"0.7.0")
 b,e:=CreateSourceWithKit(root,root,"B.c",newer);if e!=nil{t.Fatal(e)}
 data,_:=os.ReadFile(b.Path);if !strings.Contains(string(data),"/0.7.0/expanded.h"){t.Fatal(string(data))}
 again,_:=os.ReadFile(a.Path);if !bytes.Equal(raw,again){t.Fatal("old source changed")}
 if e=os.RemoveAll(pkg);e!=nil{t.Fatal(e)}
 if _,e=LoadKitSnapshot(old);e!=nil{t.Fatal("snapshot depended on installation",e)}
 header:=filepath.Join(old,"expanded.h");if e=os.WriteFile(header,[]byte("bad"),0600);e!=nil{t.Fatal(e)}
 if _,e=LoadKitSnapshot(old);e==nil{t.Fatal("tampering accepted")}
 if _,e=SaveKitSnapshot(root,k);e==nil{t.Fatal("corrupt snapshot overwritten")}
}
func TestKitValidationAndMissing(t *testing.T){
 root,e:=OpenDirectory(t.TempDir());if e!=nil{t.Fatal(e)}
 if _,e=CreateSourceWithKit(root,root,"A.c","");e==nil{t.Fatal("missing package accepted")}
 if _,e=os.Stat(filepath.Join(root,"A.c"));!os.IsNotExist(e){t.Fatal("source created on resource failure")}
 if _,e=CreateSourceWithKit(root,root,"A.cpp","");e!=nil{t.Fatal(e)}
 pkg:=kitFixture(t,"1.0.0");k,e:=LoadKitPackage(pkg);if e!=nil{t.Fatal(e)}
 if _,e=SaveKitSnapshot(root,k);e!=nil{t.Fatal(e)}
 if e=os.WriteFile(filepath.Join(pkg,"LICENSE"),[]byte("different"),0600);e!=nil{t.Fatal(e)}
 changed,e:=LoadKitPackage(pkg);if e!=nil{t.Fatal(e)}
 if _,e=SaveKitSnapshot(root,changed);e==nil{t.Fatal("same-version conflict accepted")}
 m:=k.Manifest
 m.Components[1].Requires=[]string{"read_int"};raw,_:=json.Marshal(m)
 if _,_,e=parseKitManifest(raw);e==nil{t.Fatal("cycle accepted")}
 m.Components[1].Requires=[]string{"missing"};raw,_=json.Marshal(m)
 if _,_,e=parseKitManifest(raw);e==nil{t.Fatal("unknown dependency accepted")}
 m.Components[1].Requires=[]string{};m.SystemHeaders=[]string{"../bad.h"};raw,_=json.Marshal(m)
 if _,_,e=parseKitManifest(raw);e==nil{t.Fatal("unsafe header accepted")}
}
func TestRealKitPackage(t *testing.T){
 pkg:=os.Getenv("COMPANION_TEST_KIT_PACKAGE")
 if pkg==""{t.Skip("set COMPANION_TEST_KIT_PACKAGE to built Exam Kit npm package")}
 requireCompiler(t)
 root,e:=OpenDirectory(t.TempDir());if e!=nil{t.Fatal(e)}
 sub:=filepath.Join(root,"中文 作业");if e=os.Mkdir(sub,0700);e!=nil{t.Fatal(e)}
 s,e:=CreateSourceWithKit(root,sub,"A.c",pkg);if e!=nil{t.Fatal(e)}
 k,e:=LoadKitPackage(pkg);if e!=nil{t.Fatal(e)}
 if len(k.Order)<50{t.Fatal("not real Exam Kit",len(k.Order))}
 raw,e:=os.ReadFile(s.Path);if e!=nil{t.Fatal(e)}
 raw=bytes.Replace(raw,[]byte("return 0;"),[]byte("printf(\"%d\\n\", read_int()); return 0;"),1)
 if e=os.WriteFile(s.Path,raw,0600);e!=nil{t.Fatal(e)}
 // Compile as an external editor would: the original source and its full header.
 // Companion's own minimal assembly is intentionally deferred to 04b.
 cc,e:=compilerPath("");if e!=nil{t.Fatal(e)}
 out:=filepath.Join(t.TempDir(),"program");if runtime.GOOS=="windows"{out+=".exe"}
 c:=process(context.Background(),cc,[]string{"-std=c99",s.Name,"-o",artifactArg(s.Directory,out),"-lm"},s.Directory,"",childEnv(cc),30*time.Second)
 if c.State!="ok"{t.Fatalf("full header compilation: %+v",c)}
 r:=process(context.Background(),out,nil,s.Directory,"42\n",childEnv(cc),2*time.Second)
 if r.State!="ok"||normalize(r.Stdout)!="42"{t.Fatalf("%+v",r)}
 t.Logf("real Exam Kit %s: %d declarations; external compiler passed",k.Version,len(k.Order))
}

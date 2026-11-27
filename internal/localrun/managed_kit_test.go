package localrun
import("crypto/sha256";"encoding/hex";"encoding/json";"os";"path/filepath";"runtime";"strings";"testing")
func TestManagedKitCopy(t *testing.T){
 source:=kitFixture(t,"0.6.10")
 b:=resourceBundle{Protocol:ResourceProtocol,ID:"c-exam-kit",Version:"0.6.10",Files:map[string][]byte{}}
 for _,name:=range []string{"package.json","LICENSE","preset/manifest.json","preset/reference.json","preset/CC-BY-4.0.txt"}{raw,e:=os.ReadFile(filepath.Join(source,filepath.FromSlash(name)));if e!=nil{t.Fatal(e)};b.Files[name]=raw}
 raw,_:=json.Marshal(b)
 p,cleanup,e:=importManagedKit(raw);if e!=nil{t.Fatal(e)};defer cleanup()
 if e=os.RemoveAll(source);e!=nil{t.Fatal(e)}
 if k,e:=LoadKitPackage(p);e!=nil||k.Version!="0.6.10"{t.Fatal(k.Version,e)}
 b.Version="0.6.11";raw,_=json.Marshal(b);if _,_,e=importManagedKit(raw);e==nil{t.Fatal("accepted version mismatch")}
 b.Version="0.6.10";b.Files["../outside"]=[]byte("bad");raw,_=json.Marshal(b);if _,_,e=importManagedKit(raw);e==nil{t.Fatal("accepted unknown path")}
 delete(b.Files,"../outside");delete(b.Files,"LICENSE");raw,_=json.Marshal(b);if _,_,e=importManagedKit(raw);e==nil{t.Fatal("accepted missing license")}
 cleanup();if _,e=os.Stat(p);!os.IsNotExist(e){t.Fatal("session copy retained")}
}

func TestRealManagedKit(t *testing.T){
 manager:=os.Getenv("COMPANION_TEST_MANAGER");if manager==""{t.Skip("native manager required")}
 if runtime.GOOS!="windows"&&runtime.GOOS!="darwin"{t.Skip("supported targets")}
 home:=t.TempDir();if runtime.GOOS=="windows"{t.Setenv("APPDATA",home)}else{t.Setenv("HOME",home)}
 config,e:=os.UserConfigDir();if e!=nil{t.Fatal(e)}
 base:=filepath.Join(config,"Routilet","apps","c-exam-kit")
 active:=filepath.Join(base,"activation.json")
 install:=func(version string){
  src:=kitFixture(t,version);snap:=filepath.Join(base,"release-"+version);pkg:=filepath.Join(snap,"node_modules","@routilet","c-exam-kit")
  if e:=os.MkdirAll(filepath.Join(pkg,"preset"),0700);e!=nil{t.Fatal(e)}
  // Match the manager's exported JSON and canonical integrity seal.
  b:=struct{Protocol string `json:"protocol"`;ID string `json:"id"`;Version string `json:"version"`;Files map[string][]byte `json:"files"`}{ResourceProtocol,"c-exam-kit",version,map[string][]byte{}}
  for _,name:=range []string{"package.json","LICENSE","preset/manifest.json","preset/reference.json","preset/CC-BY-4.0.txt"}{raw,e:=os.ReadFile(filepath.Join(src,filepath.FromSlash(name)));if e!=nil{t.Fatal(e)};b.Files[name]=raw;if e=os.WriteFile(filepath.Join(pkg,filepath.FromSlash(name)),raw,0600);e!=nil{t.Fatal(e)}}
  raw,_:=json.Marshal(b);sum:=sha256.Sum256(raw);seal,_:=json.Marshal(hex.EncodeToString(sum[:]));if e:=os.WriteFile(filepath.Join(snap,"resource-seal.json"),seal,0600);e!=nil{t.Fatal(e)}
  a,_:=json.Marshal(map[string]string{"Snapshot":"release-"+version,"Version":version});if e:=os.WriteFile(active,a,0600);e!=nil{t.Fatal(e)}
 }
 install("0.6.10")
 p,cleanup,e:=ResolveSessionKit(manager);if e!=nil{t.Fatal(e)};defer cleanup()
 root,e:=OpenDirectory(t.TempDir());if e!=nil{t.Fatal(e)}
 old,e:=CreateSourceWithKit(root,root,"old.c",p);if e!=nil{t.Fatal(e)}
 install("0.6.11")
 if k,e:=LoadKitPackage(p);e!=nil||k.Version!="0.6.10"{t.Fatal(k.Version,e)}
 p2,c2,e:=ResolveSessionKit(manager);if e!=nil{t.Fatal(e)};defer c2()
 next,e:=CreateSourceWithKit(root,root,"new.c",p2);if e!=nil{t.Fatal(e)}
 raw,_:=os.ReadFile(next.Path);if !strings.Contains(string(raw),"/0.6.11/expanded.h"){t.Fatal(string(raw))}
 if e=os.Remove(active);e!=nil{t.Fatal(e)}
 missing,c3,e:=ResolveSessionKit(manager);defer c3();if e==nil||missing!=""{t.Fatal("removed resource accepted")}
 if _,e=CreateSourceWithKit(root,root,"plain.c",missing);e!=nil{t.Fatal(e)}
 raw,_=os.ReadFile(old.Path);if !strings.Contains(string(raw),"/0.6.10/expanded.h"){t.Fatal(string(raw))}
 if _,e=LoadKitSnapshot(filepath.Join(root,".companion","resources","0.6.10"));e!=nil{t.Fatal(e)}
}

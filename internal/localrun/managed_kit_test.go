package localrun
import("encoding/json";"os";"path/filepath";"testing")
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

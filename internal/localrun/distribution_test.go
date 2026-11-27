package localrun
import("encoding/json";"os";"path/filepath";"testing")
func TestInstalledKit(t *testing.T){
 base:=t.TempDir()
 // pnpm real path: <store>/node_modules/@routilet/companion/bin/...
 modules:=filepath.Join(base,".pnpm","companion","node_modules")
 pkg:=filepath.Join(modules,"@routilet","companion")
 exe:=filepath.Join(pkg,"bin","darwin-arm64","Companion.app","Contents","MacOS","companion")
 if e:=os.MkdirAll(filepath.Dir(exe),0700);e!=nil{t.Fatal(e)}
 if e:=os.WriteFile(exe,[]byte("fixture"),0700);e!=nil{t.Fatal(e)}
 raw,_:=json.Marshal(map[string]any{"name":PackageName,"dependencies":map[string]string{KitPackageName:"0.6.10"}})
 if e:=os.WriteFile(filepath.Join(pkg,"package.json"),raw,0600);e!=nil{t.Fatal(e)}
 original:=kitFixture(t,"0.6.10")
 target:=filepath.Join(modules,"@routilet","c-exam-kit")
 if e:=os.Rename(original,target);e!=nil{t.Fatal(e)}
 got,e:=InstalledKit(exe);if e!=nil{t.Fatal(e)}
 want,e:=filepath.EvalSymlinks(target);if e!=nil||got!=want{t.Fatal(got,want,e)}
 if e=os.RemoveAll(target);e!=nil{t.Fatal(e)}
 if _,e=InstalledKit(exe);e==nil{t.Fatal("accepted missing dependency")}
}

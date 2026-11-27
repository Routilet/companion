package ojadapter

import (
 "context"
 "crypto/sha256"
 "encoding/hex"
 "encoding/json"
 "os"
 "path/filepath"
 "runtime"
 "strings"
 "testing"
)

func TestManagedDoesNotFallBackToPreview(t *testing.T){
 a:=NewManagedAccounts(filepath.Join(t.TempDir(),"missing"),accountFixture(t))
 a.store=&memoryStore{}
 defer a.Close()
 if a.Status().Available||a.session!=nil{t.Fatal("unexpected startup")}
 if v:=a.CheckAvailability(context.Background());v.Available||!strings.Contains(v.UnavailableReason,"Routilet"){t.Fatal(v)}
 if _,e:=a.Login(context.Background(),"42","test-secret",false);e==nil{t.Fatal("fell back to preview")}
 if a.session!=nil{t.Fatal("preview process started")}
 b:=NewManagedAccounts("",accountFixture(t));b.store=&memoryStore{};defer b.Close()
 if !b.Status().Available{t.Fatal("explicit development path rejected")}
}
func TestRealManagedAdapter(t *testing.T){
 manager,adapter:=os.Getenv("COMPANION_TEST_MANAGER"),os.Getenv("COMPANION_TEST_ADAPTER")
 if manager==""||adapter==""{t.Skip("native manager and adapter required")}
 if runtime.GOOS!="windows"&&runtime.GOOS!="darwin"{t.Skip("supported platforms only")}
 home:=t.TempDir()
 if runtime.GOOS=="windows"{t.Setenv("APPDATA",home)}else{t.Setenv("HOME",home)}
 config,e:=os.UserConfigDir();if e!=nil{t.Fatal(e)}
 base:=filepath.Join(config,"Routilet","apps","accoding-adapter")
 snap:=filepath.Join(base,"release-test")
 pkg:=filepath.Join(snap,"node_modules","@routilet","accoding-adapter")
 if e=os.MkdirAll(pkg,0700);e!=nil{t.Fatal(e)}
 raw,e:=os.ReadFile(adapter);if e!=nil{t.Fatal(e)}
 platform,name:="darwin-arm64","adapter"
 if runtime.GOOS=="windows"{platform,name="win32-x64","adapter.exe"}
 if e=os.WriteFile(filepath.Join(pkg,name),raw,0700);e!=nil{t.Fatal(e)}
 sum:=sha256.Sum256(raw)
 manifest:=map[string]any{"schemaVersion":1,"version":"0.1.0-beta.1","adapter":"accoding","protocol":Protocol,"platforms":map[string]any{platform:map[string]string{"path":name,"sha256":hex.EncodeToString(sum[:])}}}
 b,_:=json.Marshal(manifest)
 if e=os.WriteFile(filepath.Join(pkg,"manifest.json"),b,0600);e!=nil{t.Fatal(e)}
 active:=filepath.Join(base,"activation.json")
 if e=os.WriteFile(active,[]byte(`{"Snapshot":"release-test","Version":"0.1.0-beta.1"}`),0600);e!=nil{t.Fatal(e)}
 a:=NewManagedAccounts(manager,"");a.store=&memoryStore{};defer a.Close()
 if a.Status().Available||a.session!=nil{t.Fatal("unexpected startup process")}
 if v:=a.CheckAvailability(context.Background());!v.Available||a.session!=nil{t.Fatal(v)}
 if e=a.ensure(context.Background());e!=nil{t.Fatal(e)}
 var state AccountState
 if e=a.session.Call(context.Background(),"account",map[string]bool{"refresh":false},&state);e!=nil||state.State!="signed_out"{t.Fatal(state,e)}
 runs:=filepath.Join(snap,".runs")
 if entries,e:=os.ReadDir(runs);e!=nil||len(entries)!=1{t.Fatal("missing running marker",entries,e)}
 // Simulate removal of activation while a leased session is still running.
 if e=os.Remove(active);e!=nil{t.Fatal(e)}
 if e=a.session.Call(context.Background(),"account",map[string]bool{"refresh":false},&state);e!=nil{t.Fatal(e)}
 if _,e=a.Logout();e!=nil{t.Fatal(e)}
 if entries,e:=os.ReadDir(runs);e!=nil||len(entries)!=0{t.Fatal("normal EOF left a marker",entries,e)}
 if a.CheckAvailability(context.Background()).Available{t.Fatal("removed activation still available")}
}

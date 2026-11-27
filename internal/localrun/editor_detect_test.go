package localrun
import ("os";"path/filepath";"testing")
func TestDetectEditorSpecs(t *testing.T){
 p:=filepath.Join(t.TempDir(),"editor.exe");if e:=os.WriteFile(p,[]byte("fixture"),0700);e!=nil{t.Fatal(e)}
 got:=detectEditorSpecs([]editorSpec{{"missing","",[]string{"relative.exe",p+".missing"}},{"fixture","",[]string{p}}})
 if len(got)!=1||got[0].Path!=p||got[0].Name!="fixture"{t.Fatalf("%+v",got)}
}

package localrun
import("bytes";"os";"path/filepath";"strings";"testing")
func TestKitSwitchPlanAndApply(t *testing.T){
 root,e:=OpenDirectory(t.TempDir());if e!=nil{t.Fatal(e)}
 old:=kitFixture(t,"0.6.10");newer:=kitFixture(t,"0.6.11")
 a,e:=CreateSourceWithKit(root,root,"A.c",old);if e!=nil{t.Fatal(e)}
 b,e:=CreateSourceWithKit(root,root,"B.c",old);if e!=nil{t.Fatal(e)}
 c,e:=CreateSourceWithKit(root,root,"C.c","");if e!=nil{t.Fatal(e)}
 before,e:=os.ReadFile(b.Path);if e!=nil{t.Fatal(e)};modified:=bytes.Replace(before,[]byte("return 0;"),[]byte("return read_int();"),1)
 if e=os.WriteFile(b.Path,modified,0600);e!=nil{t.Fatal(e)}
 k,e:=LoadKitPackage(newer);if e!=nil{t.Fatal(e)}
 p,e:=PlanKitSwitch(root,[]string{a.Path,b.Path,c.Path},k);if e!=nil||p.Rows[0].Kind!="template"||p.Rows[1].Kind!="modified"||p.Rows[2].Kind!="attach"{t.Fatal(p,e)}
 if e=ApplyKitChange(root,p.Rows[0],k);e!=nil{t.Fatal(e)}
 raw,_:=os.ReadFile(a.Path);if !strings.Contains(string(raw),"/0.6.11/expanded.h"){t.Fatal(string(raw))}
 unchanged,_:=os.ReadFile(b.Path);if !bytes.Equal(unchanged,modified){t.Fatal("modified file changed")}
 if _,e=LoadKitSnapshot(filepath.Join(root,".companion","resources","0.6.10"));e!=nil{t.Fatal(e)}
 if e=os.WriteFile(b.Path,append(modified,[]byte("\n/* external edit */\n")...),0600);e!=nil{t.Fatal(e)}
 if e=ApplyKitChange(root,p.Rows[1],k);e==nil{t.Fatal("stale preview accepted")}
 p,e=PlanKitSwitch(root,[]string{b.Path,c.Path},k);if e!=nil{t.Fatal(e)}
 if e=ApplyKitChange(root,p.Rows[0],k);e!=nil{t.Fatal(e)}
 raw,_=os.ReadFile(b.Path);if !strings.Contains(string(raw),"return read_int();")||!strings.Contains(string(raw),"external edit"){t.Fatal(string(raw))}
 if e=ApplyKitChange(root,p.Rows[1],k);e!=nil{t.Fatal(e)}
 raw,_=os.ReadFile(c.Path);if !strings.Contains(string(raw),"/0.6.11/expanded.h"){t.Fatal(string(raw))}
}

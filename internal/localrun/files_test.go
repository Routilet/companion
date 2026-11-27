package localrun

import (
 "context"
 "os"
 "path/filepath"
 "testing"
)
func TestFileTreeAndCreate(t *testing.T){
 p:=source(t,"int main(void){return 0;}")
 root,e:=OpenDirectory(filepath.Dir(p));if e!=nil{t.Fatal(e)}
 for _,n:=range []string{".companion",".accoding","作业"}{if e=os.Mkdir(filepath.Join(root,n),0700);e!=nil{t.Fatal(e)}}
 if e=os.WriteFile(filepath.Join(root,"notes.txt"),[]byte("note"),0600);e!=nil{t.Fatal(e)}
 list,e:=ListDirectory(root,root);if e!=nil||len(list)!=2||!list[0].Directory{t.Fatalf("%+v %v",list,e)}
 if _,e=Within(root,filepath.Dir(root));e==nil{t.Fatal("outside root accepted")}
 if _,e=Within(root,filepath.Join(root,".accoding"));e==nil{t.Fatal("hidden accepted")}
 s,e:=CreateSource(root,filepath.Join(root,"作业"),"A.cpp");if e!=nil{t.Fatal(e)}
 if !isCPP(s.Path){t.Fatal(s)}
 if _,e=CreateSource(root,s.Directory,s.Name);e==nil{t.Fatal("overwrote source")}
 if _,e=CreateSource(root,s.Directory,"../B.c");e==nil{t.Fatal("traversal accepted")}
}
func TestRenameAndRestore(t *testing.T){
 p:=source(t,"int main(void){return 0;}")
 root,e:=OpenDirectory(filepath.Dir(p));if e!=nil{t.Fatal(e)}
 p=filepath.Join(root,filepath.Base(p))
 cs,e:=SaveCases(p,[]Case{{"abc",str("xyz")}},"missing");if e!=nil{t.Fatal(e)}
 s,e:=RenameSource(root,p,"B.c");if e!=nil{t.Fatal(e)}
 if _,e=os.Stat(p);!os.IsNotExist(e){t.Fatal("old source remains")}
 got,e:=LoadCases(s.Path);if e!=nil||got.Revision!=cs.Revision{t.Fatalf("%+v %v",got,e)}
 q:=filepath.Join(root,"C.c");if e=os.Rename(s.Path,q);e!=nil{t.Fatal(e)}
 list,e:=ListOrphans(q);if e!=nil||len(list)!=1||list[0].Name!="B.c"{t.Fatalf("%+v %v",list,e)}
 if _,e=CreateSource(root,root,"B.c");e==nil{t.Fatal("orphan collision accepted")}
 restored,e:=RestoreOrphan(q,"B.c","missing");if e!=nil||len(restored.Cases)!=1{t.Fatalf("%+v %v",restored,e)}
 if _,e=os.Stat(filepath.Join(root,".companion","cases","B.c.json"));e!=nil{t.Fatal("original data not retained")}
 if _,e=RenameSource(root,q,"B.c");e==nil{t.Fatal("orphan target overwritten")}
 if _,e=RestoreOrphan(q,"B.c",restored.Revision);e==nil{t.Fatal("existing cases replaced")}
}
func TestCPP(t *testing.T){
 p:=source(t,"")
 q:=filepath.Join(filepath.Dir(p),"A.cpp")
 if e:=os.WriteFile(q,[]byte("#include <iostream>\nint main(){std::cout << 42;}\n"),0600);e!=nil{t.Fatal(e)}
 r,e:=Run(context.Background(),q,"","")
 if e!=nil||r.State!="ran"||r.Execution==nil||r.Execution.Stdout!="42"{t.Fatalf("%+v %v",r,e)}
}

func TestImportAppendAndSeparateExtensions(t *testing.T){
 p:=source(t,"int main(void){return 0;}")
 root,e:=OpenDirectory(filepath.Dir(p));if e!=nil{t.Fatal(e)}
 p=filepath.Join(root,filepath.Base(p))
 q,e:=CreateSource(root,root,"A.cpp");if e!=nil{t.Fatal(e)}
 cs,e:=SaveCases(p,[]Case{{"one",nil}},"missing");if e!=nil{t.Fatal(e)}
 cp,_:=casePath(p)
 got,e:=ImportCases(q.Path,cp,"missing",false);if e!=nil||len(got.Cases)!=1{t.Fatalf("%+v %v",got,e)}
 if _,e=ImportCases(q.Path,cp,got.Revision,false);e==nil{t.Fatal("implicit overwrite")}
 got,e=ImportCases(q.Path,cp,got.Revision,true);if e!=nil||len(got.Cases)!=2{t.Fatalf("%+v %v",got,e)}
 original,e:=LoadCases(p);if e!=nil||original.Revision!=cs.Revision{t.Fatal("original changed")}
}

func TestCreateFolderNoOverwrite(t *testing.T){
 root:=t.TempDir();p,e:=CreateFolder(root,root,"作业 1");if e!=nil{t.Fatal(e)}
 if p!=filepath.Join(root,"作业 1"){t.Fatal(p)}
 for _,name:=range []string{"作业 1","../escape",".hidden","CON","trailing."}{if _,e:=CreateFolder(root,root,name);e==nil{t.Fatalf("accepted %q",name)}}
}

package localrun

import (
 "encoding/json"
 "os"
 "path/filepath"
 "strings"
 "testing"
)
func referenceFixture(t *testing.T,version string)string{
 t.Helper();pkg:=kitFixture(t,version)
 r:=Reference{SchemaVersion:1,Version:version,Notes:"使用约定",Categories:[]ReferenceCategory{
  {Title:"输入与输出",Entries:[]ReferenceEntry{
   {ID:"read_number",Title:"读一个数",Description:"读取整数 浮点数",Example:"int n = read_int();",Keywords:[]string{"数字"},Related:[]string{"other"},Signatures:[]string{"int read_int(void);","double read_double(void);"},Variants:[]ReferenceVariant{
    {Key:"int",Label:"int",Symbols:[]string{"read_int"},Signatures:[]string{"int read_int(void);"},Example:"int n = read_int();"},
    {Key:"double",Label:"double",Symbols:[]string{"read_double"},Signatures:[]string{"double read_double(void);"},Example:"double n = read_double();"},
   }},
   {ID:"other",Title:"输出",Description:"打印",Example:"puts(\"x\");",Signatures:[]string{"void output(void);"}},
  }},
 },Sections:[]ReferenceSection{{Title:"输入与输出",Children:[]ReferenceSection{{Title:"数字读写",Entries:[]string{"read_number","other"}}}}}}
 b,e:=json.Marshal(r);if e!=nil{t.Fatal(e)}
 if e=os.WriteFile(filepath.Join(pkg,"preset","reference.json"),b,0600);e!=nil{t.Fatal(e)}
 return pkg
}
func TestManualVersionAndOffline(t *testing.T){
 old,newer:=referenceFixture(t,"0.6.10"),referenceFixture(t,"0.7.0")
 s:=kitSource(t,old,"#define UNFINISHED 1\nint main(void){")
 m,e:=LoadManual(s.Path,newer);if e!=nil||m.Origin!="snapshot"||m.Reference.Version!="0.6.10"{t.Fatalf("%+v %v",m,e)}
 if e=os.RemoveAll(old);e!=nil{t.Fatal(e)}
 m,e=LoadManual(s.Path,"");if e!=nil||m.Reference.Version!="0.6.10"{t.Fatal(e)}
 m,e=LoadManual("",newer);if e!=nil||m.Origin!="installed"||m.Reference.Version!="0.7.0"{t.Fatal(e)}
 p:=source(t,"int main(void){return 0;}")
 m,e=LoadManual(p,newer);if e!=nil||m.Origin!="installed"{t.Fatal(e)}
 header:=filepath.Join(s.Directory,".companion","resources","0.6.10","expanded.h")
 if e=os.WriteFile(header,[]byte("bad"),0600);e!=nil{t.Fatal(e)}
 if _,e=LoadManual(s.Path,newer);e==nil{t.Fatal("silently fell back to installed version")}
}
func TestReferenceSearchAndExample(t *testing.T){
 m,e:=LoadManual("",referenceFixture(t,"0.6.10"));if e!=nil{t.Fatal(e)}
 hits:=SearchReference(m.Reference,"read_double")
 if len(hits)!=1||hits[0].ID!="read_number"||hits[0].Variant!="double"{t.Fatal(hits)}
 if len(SearchReference(m.Reference,"数字 读取"))!=1||len(SearchReference(m.Reference,"not found"))!=0{t.Fatal("search")}
 ex,e:=m.Example("read_number","double");if e!=nil||ex!="double n = read_double();"{t.Fatal(ex,e)}
 if _,e=m.Example("read_number","unknown");e==nil{t.Fatal("invalid variant")}
 if _,e=m.Example("other","int");e==nil{t.Fatal("invalid variant on plain entry")}
}
func TestMalformedReference(t *testing.T){
 m,e:=LoadManual("",referenceFixture(t,"0.6.10"));if e!=nil{t.Fatal(e)}
 r:=m.Reference;r.Categories[0].Entries[0].Related=[]string{"unknown"}
 raw,_:=json.Marshal(r);if _,e=ParseReference(raw,r.Version);e==nil{t.Fatal("bad related reference accepted")}
 r=m.Reference;r.Categories[0].Entries[0].Related=[]string{"other"};r.Sections=[]ReferenceSection{{Title:"bad",Entries:[]string{"missing"}}}
 raw,_=json.Marshal(r);if _,e=ParseReference(raw,r.Version);e==nil{t.Fatal("bad navigation accepted")}
}
func TestActualKitReference(t *testing.T){
 pkg:=os.Getenv("COMPANION_TEST_KIT_PACKAGE");if pkg==""{t.Skip("real package fixture not provided")}
 m,e:=LoadManual("",pkg);if e!=nil{t.Fatal(e)}
 count:=0;for _,c:=range m.Reference.Categories{count+=len(c.Entries)}
 if m.Reference.SchemaVersion==2{count=len(m.Reference.AllEntries())}
 if count<50||(m.Reference.SchemaVersion==1&&len(m.Reference.Sections)!=5)||m.Reference.Notes==""{t.Fatal("unexpected actual reference",count)}
 hits:=SearchReference(m.Reference,"read_double")
 if len(hits)==0||hits[0].ID!="read_number"||hits[0].Variant!="double"{t.Fatal(hits)}
 ex,e:=m.Example(hits[0].ID,hits[0].Variant);if e!=nil||!strings.Contains(ex,"read_double"){t.Fatal(ex,e)}
 t.Logf("actual reference %s: %d entries, %d sections",m.Reference.Version,count,len(m.Reference.Sections))
}

func TestFlatReferenceCompatibility(t *testing.T){
 m,e:=LoadManual("",referenceFixture(t,"0.6.10"));if e!=nil{t.Fatal(e)}
 r:=m.Reference;entry:=r.Categories[0].Entries[0];entry.Tags=[]string{"数字","输入"};entry.Summary="读取数字";entry.Keywords=[]string{"从标准输入"};entry.Related=nil
 r=Reference{SchemaVersion:2,Version:r.Version,Notes:r.Notes,Tags:[]string{"数字","输入"},Entries:[]ReferenceEntry{entry}}
 raw,_:=json.Marshal(r);got,e:=ParseReference(raw,r.Version);if e!=nil{t.Fatal(e)}
 if len(got.AllEntries())!=1||len(SearchReference(got,"从标准输入"))!=1{t.Fatal("flat not preferred")}
 hits:=SearchReference(got,"read_double");if len(hits)!=1||hits[0].Variant!="double"{t.Fatal(hits)}
 r.Entries[0].Tags=[]string{"不存在"};raw,_=json.Marshal(r);if _,e=ParseReference(raw,r.Version);e==nil{t.Fatal("unknown flat tag accepted")}
}

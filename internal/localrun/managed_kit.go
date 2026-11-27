package localrun

import (
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "strings"
 "time"
)

const ResourceProtocol="routilet-resource/1"
type resourceBundle struct{
 Protocol,ID,Version string
 Files map[string][]byte
}
func importManagedKit(raw []byte)(string,func(),error){
 noop:=func(){}
 var b resourceBundle
 if len(raw)>4<<20||json.Unmarshal(raw,&b)!=nil||b.Protocol!=ResourceProtocol||b.ID!="c-exam-kit"||!kitVersion.MatchString(b.Version){return "",noop,errors.New("Exam Kit 资源协议或版本无效")}
 required:=map[string]bool{"package.json":false,"LICENSE":false,"preset/manifest.json":false,"preset/reference.json":false,"preset/CC-BY-4.0.txt":false}
 total:=0
 for name,data:=range b.Files{
  if _,ok:=required[name];!ok&&name!="preset/manual.zh-CN.md"{return "",noop,errors.New("未知资源文件")}
  total+=len(data);if len(data)==0||total>2<<20{return "",noop,errors.New("资源大小无效")}
  if _,ok:=required[name];ok{required[name]=true}
 }
 for _,present:=range required{if !present{return "",noop,errors.New("资源文件缺失")}}
 root,e:=os.MkdirTemp("","companion-kit-");if e!=nil{return "",noop,e}
 cleanup:=func(){_ = os.RemoveAll(root)}
 fail:=func(e error)(string,func(),error){cleanup();return "",noop,e}
 if e=os.Mkdir(filepath.Join(root,"preset"),0700);e!=nil{return fail(e)}
 for name,data:=range b.Files{if e=os.WriteFile(filepath.Join(root,filepath.FromSlash(name)),data,0600);e!=nil{return fail(e)}}
 k,e:=LoadKitPackage(root);if e!=nil{return fail(e)}
 if k.Version!=b.Version{return fail(errors.New("Exam Kit 资源版本不一致"))}
 return root,cleanup,nil
}
// Resolve a read-only resource copy at session start; no shared installation
// path survives an update or uninstall. Never silently fall back if managed.
func ResolveSessionKit(manager string)(string,func(),error){
 noop:=func(){}
 if manager==""{
  if p:=os.Getenv(KitPreviewEnv);p!=""{if _,e:=LoadKitPackage(p);e!=nil{return "",noop,e};return p,noop,nil}
  return "",noop,errors.New("请通过 Routilet 安装并启用 C Exam Kit 后重新打开 Companion")
 }
 if !filepath.IsAbs(manager){return "",noop,errors.New("Routilet 调用入口无效")}
 p:=process(context.Background(),manager,[]string{"--resource-probe"},filepath.Dir(manager),"",nil,5*time.Second)
 if p.State!="ok"||strings.TrimSpace(p.Stdout)!=ResourceProtocol{return "",noop,errors.New("请更新 Routilet 至支持资源协议的版本")}
 p=process(context.Background(),manager,[]string{"--resource-export","c-exam-kit"},filepath.Dir(manager),"",nil,15*time.Second)
 if p.State!="ok"{return "",noop,fmt.Errorf("C Exam Kit 不可用，请在 Routilet 安装、启用或修复后重新打开 Companion: %s %s",p.State,p.Stderr)}
 return importManagedKit([]byte(p.Stdout))
}

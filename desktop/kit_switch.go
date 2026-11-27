package main
import("errors";"fmt";"path/filepath";"strings";"github.com/Routilet/companion/internal/localrun")
type kitPending struct{Root,Token string;Plan localrun.KitPlan}
type KitPreview struct{Token string `json:"token"`;Plan localrun.KitPlan `json:"plan"`}
type KitApplied struct{Changed []string `json:"changed"`;Failed []string `json:"failed"`}
func(d *Desktop) PreviewKitSwitch(path string,group []string)(KitPreview,error){
 d.mu.Lock();defer d.mu.Unlock();d.kitPending=nil
 if d.running||d.observerDone!=nil||d.root==""{return KitPreview{},errors.New("请结束运行或观察")}
 paths:=[]string{}
 if len(group)>0{paths=append(paths,group...)}else{
 p,e:=localrun.Within(d.root,path);if e!=nil{return KitPreview{},e}
 if strings.EqualFold(filepath.Ext(p),".c"){paths=append(paths,p)}else{
 queue:=[]string{p};for n:=0;len(queue)>0;n++{if n>=4096||len(paths)>512||len(queue)>4096{return KitPreview{},errors.New("目录过大，请缩小范围")};dir:=queue[0];queue=queue[1:];entries,e:=localrun.ListDirectory(d.root,dir);if e!=nil{return KitPreview{},e};for _,v:=range entries{if v.Directory{queue=append(queue,v.Path)}else if strings.EqualFold(filepath.Ext(v.Path),".c"){paths=append(paths,v.Path)}}}
 }}
 pkg,cleanup,e:=d.installedKit();defer cleanup();if e!=nil{return KitPreview{},e};k,e:=localrun.LoadKitPackage(pkg);if e!=nil{return KitPreview{},e};plan,e:=localrun.PlanKitSwitch(d.root,paths,k);if e!=nil{return KitPreview{},e};d.ojSequence++;token:=fmt.Sprint(d.ojSequence);d.kitPending=&kitPending{d.root,token,plan};return KitPreview{token,plan},nil
}
func(d *Desktop) ApplyKitSwitch(token string,paths []string)(KitApplied,error){
 d.mu.Lock();defer d.mu.Unlock();out:=KitApplied{[]string{},[]string{}};pending:=d.kitPending;d.kitPending=nil
 if d.running||d.observerDone!=nil||pending==nil||pending.Root!=d.root||pending.Token!=token||len(paths)==0||len(paths)>512{return out,errors.New("资源切换预览已失效")}
 wanted:=map[string]bool{};for _,p:=range paths{wanted[p]=true};rows:=[]localrun.KitChange{};for _,r:=range pending.Plan.Rows{if wanted[r.Path]{delete(wanted,r.Path);if r.Kind!="template"&&r.Kind!="modified"&&r.Kind!="attach"{return out,errors.New("无效切换选择")};rows=append(rows,r)}};if len(wanted)>0{return out,errors.New("无效切换选择")}
 pkg,cleanup,e:=d.installedKit();defer cleanup();if e!=nil{return out,e};k,e:=localrun.LoadKitPackage(pkg);if e!=nil{return out,e};if k.Fingerprint!=pending.Plan.Fingerprint||k.Version!=pending.Plan.Version{return out,errors.New("已安装版本变化，请重新预览")}
 for _,r:=range rows{if e=localrun.ApplyKitChange(d.root,r,k);e!=nil{out.Failed=append(out.Failed,r.Path+"："+e.Error());continue};out.Changed=append(out.Changed,r.Path);delete(d.last,r.Path)}
 d.prepared=nil;d.ojSubmit=nil;d.manual=nil;return out,nil
}

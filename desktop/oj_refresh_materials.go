package main

import (
 "context"
 "errors"
 "fmt"
 "path/filepath"
 "strings"
 "time"

 "github.com/Routilet/companion/internal/localrun"
 "github.com/Routilet/companion/internal/ojadapter"
)

type OJRefreshMaterialsResult struct {
 Updated []string `json:"updated"`
 Unchanged []string `json:"unchanged"`
 Skipped []string `json:"skipped"`
 Failed []string `json:"failed"`
 Backups []string `json:"backups"`
 Missing int `json:"missing"`
}
type materialSnapshot struct {
 path string
 binding ojadapter.Binding
 doc *localrun.Document
 cases localrun.CaseSet
}

// Refresh only existing, locally bound files in the selected contest group.
// Explicit confirmation is provided by the desktop UI before this call.
func(d *Desktop) OJRefreshMaterials(paths []string)(OJRefreshMaterialsResult,error){
 out:=OJRefreshMaterialsResult{Updated:[]string{},Unchanged:[]string{},Skipped:[]string{},Failed:[]string{},Backups:[]string{}}
 if len(paths)==0||len(paths)>512{return out,errors.New("请选择当前比赛")}
 if d.accounts==nil{return out,errors.New("请先登录 Accoding")}
 account:=d.accounts.Status()
 if !account.Available||account.ID==""{return out,errors.New("请先登录 Accoding")}
 d.mu.Lock()
 root:=d.root;running:=d.running;d.mu.Unlock()
 if running||root==""{return out,errors.New("请先结束当前操作")}
 view,e:=d.OJWorkspace();if e!=nil{return out,e}
 items:=map[string]OJItem{};for _,v:=range view.Items{items[v.Path]=v}
 seen:=map[string]bool{};snapshots:=[]materialSnapshot{};docOwners:=map[string]int{}
 contest,group,link:="","",""
 for _,path:=range paths{
  item,ok:=items[path];if !ok||seen[path]||item.Binding.Target.Kind!="contest"{return out,errors.New("请选择同一比赛的本地题目")}
  seen[path]=true
  if contest==""{contest=item.Binding.Target.ContestID;group=item.Group;link=item.Binding.SourceURL}
  if item.Binding.Target.ContestID!=contest||item.Group!=group||item.Binding.SourceURL!=link{return out,errors.New("不能同时更新不同比赛")}
  if _,e=localrun.Within(root,path);e!=nil{return out,e}
  doc,err:=localrun.ReadDocument(root,path);if err!=nil{out.Failed=append(out.Failed,path+"："+err.Error());continue}
  cs,err:=localrun.LoadCases(path);if err!=nil{out.Failed=append(out.Failed,path+"："+err.Error());continue}
  if doc!=nil{docOwners[filepath.Join(filepath.Dir(path),doc.Name)]++}
  snapshots=append(snapshots,materialSnapshot{path,item.Binding,doc,cs})
 }
 ctx,cancel:=context.WithTimeout(context.Background(),90*time.Second);defer cancel()
 remote,e:=d.accounts.Import(ctx,link);if e!=nil{return out,e}
 if e=remote.Validate(account.ID);e!=nil{return out,e}
 problems:=map[ojadapter.Target]ojadapter.Problem{}
 for _,p:=range remote.Problems{
  if p.Target.Kind!="contest"||p.Target.ContestID!=contest{return out,errors.New("返回的比赛与当前比赛不一致")}
  problems[p.Target]=p
 }
 localTargets:=map[ojadapter.Target]bool{};for path:=range seen{localTargets[items[path].Binding.Target]=true}
 for target:=range problems{if !localTargets[target]{out.Missing++}}
 d.mu.Lock();defer d.mu.Unlock()
 a:=d.accounts.Status()
 if d.running||d.root!=root||a.ID!=account.ID||a.State!="signed_in"||!a.Available{return out,errors.New("工作区或账号已变化，请重新更新")}
 for _,s:=range snapshots{
  p,ok:=problems[s.binding.Target];if !ok{out.Skipped=append(out.Skipped,s.path+"：比赛中已无此题，保留本地资料");continue}
  if _,e=localrun.Within(root,s.path);e!=nil{out.Failed=append(out.Failed,s.path+"："+e.Error());continue}
  b,err:=ojadapter.LoadBinding(s.path);if err!=nil||b==nil||*b!=s.binding{out.Failed=append(out.Failed,s.path+"：题目关联已变化");continue}
  markdown:=p.Markdown()
  // Only replace generated statements, never an unrelated user document.
  if s.doc!=nil&&(docOwners[filepath.Join(filepath.Dir(s.path),s.doc.Name)]>1||!strings.HasPrefix(s.doc.Content,"# "+b.Label+" · ")||!strings.Contains(s.doc.Content,"\n\n来源：<"+b.SourceURL+">\n\n")){
   markdown="";out.Skipped=append(out.Skipped,s.path+"：保留自定义或共享文档")
  }
  samples:=[]localrun.Case{}
  if !p.Special&&len(p.Warnings)==0{
   for _,sample:=range p.Samples{expected:=sample.Expected;samples=append(samples,localrun.Case{Input:sample.Input,Expected:&expected})}
  }else{out.Skipped=append(out.Skipped,s.path+"：在线样例无法可靠更新，保留原样例")}
  changed,backup,err:=localrun.RefreshProblemData(root,s.path,s.doc,s.cases,markdown,samples)
  if backup!=""{out.Backups=append(out.Backups,backup)}
  if changed{delete(d.last,s.path);if d.selected.Path==s.path{d.prepared=nil}}
  if err!=nil{out.Failed=append(out.Failed,fmt.Sprintf("%s：%v",s.path,err));continue}
  if changed{out.Updated=append(out.Updated,s.path)}else{out.Unchanged=append(out.Unchanged,s.path)}
 }
 return out,nil
}

package main
import("context";"errors";"path/filepath";"time";"github.com/Routilet/companion/internal/localrun";"github.com/Routilet/companion/internal/ojadapter")
type OJItem struct{Path string `json:"path"`;Directory string `json:"directory"`;Group string `json:"group"`;Binding ojadapter.Binding `json:"binding"`}
type OJWorkspaceView struct{Items []OJItem `json:"items"`;Limited bool `json:"limited"`}
// A bounded local projection only; ordinary file browsing does not call it.
func(d *Desktop)OJWorkspace()(OJWorkspaceView,error){
 out:=OJWorkspaceView{Items:[]OJItem{}}
 if d.accounts==nil||!d.accounts.Status().Available{return out,nil}
 d.mu.Lock();root:=d.root;d.mu.Unlock();if root==""{return out,nil}
 queue:=[]string{root};dirs,sources:=0,0
 for len(queue)>0&&dirs<4096&&sources<4096{
  dir:=queue[0];queue=queue[1:];dirs++
  entries,e:=localrun.ListDirectory(root,dir);if e!=nil{if dir==root{return out,e};continue}
  for _,entry:=range entries{if entry.Directory{if len(queue)<4096{queue=append(queue,entry.Path)}else{out.Limited=true};continue}
   sources++;if sources>4096{out.Limited=true;break}
   b,e:=ojadapter.LoadBinding(entry.Path);if e!=nil||b==nil{continue}
   group:=dir
   if b.Target.Kind=="contest"&&(filepath.Base(dir)==b.Label||filepath.Base(dir)=="q"+b.Label){group=filepath.Dir(dir)}
   rel,_:=filepath.Rel(root,group);if rel=="."{rel=filepath.Base(root)}
   out.Items=append(out.Items,OJItem{entry.Path,dir,rel,*b});if len(out.Items)>=512{out.Limited=true;return out,nil}
  }
 }
 if len(queue)>0{out.Limited=true}
 return out,nil
}
func(d *Desktop)OJRecentContests()([]ojadapter.RecentContest,error){
 a,ok:=d.accounts.(interface{RecentContests(context.Context)([]ojadapter.RecentContest,error)});if !ok{return nil,errors.New("比赛列表不可用，请输入链接")}
 ctx,cancel:=context.WithTimeout(context.Background(),20*time.Second);defer cancel();return a.RecentContests(ctx)
}
func(d *Desktop)ReadLocalDocument()(*localrun.Document,error){
 d.mu.Lock();root,path:=d.root,d.selected.Path;d.mu.Unlock()
 if path==""{return nil,nil};return localrun.ReadDocument(root,path)
}
func(d *Desktop)OJOpenContestList()error{return localrun.OpenOJLink("https://accoding.buaa.edu.cn/contest/index")}

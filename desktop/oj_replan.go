package main

import ("errors";"fmt";"strings";"github.com/Routilet/companion/internal/localrun";"github.com/Routilet/companion/internal/ojadapter")

// Replan from the same in-memory remote result; no second OJ request.
func(d *Desktop)OJReplanImport(token,name string)(OJPreview,error){
 d.mu.Lock();defer d.mu.Unlock();old:=d.ojPending
 if d.running||old==nil||old.token!=token||old.root!=d.root||!old.plan.CanRename{return OJPreview{},errors.New("导入计划不可重命名或已失效")}
 if a:=d.accounts.Status();!a.Available||a.State!="signed_in"||a.ID!=old.plan.AccountID{return OJPreview{},errors.New("账号状态已变化，请重新导入")}
 if _,e:=localrun.Within(d.root,old.plan.Directory);e!=nil{return OJPreview{},e}
 if strings.TrimSpace(name)==""{return OJPreview{},errors.New("请输入比赛文件夹名称")}
 plan,e:=ojadapter.BuildArchivePlanNamed(old.result,old.plan.Directory,old.ext,"",name,func(path string)(string,bool,error){s,e:=localrun.LoadCases(path);return s.Revision,len(s.Cases)==0,e})
 if e!=nil{return OJPreview{},e}
 d.ojSequence++;old.token=fmt.Sprint(d.ojSequence);old.plan=plan
 return OJPreview{old.token,plan},nil
}

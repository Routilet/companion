package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/Routilet/companion/internal/localrun"
	"github.com/Routilet/companion/internal/ojadapter"
	"path/filepath"
 "os"
 "strings"
)

// Account interface lets the desktop flow be tested without credentials or OJ access.
type ojAccountService interface {
 Status() ojadapter.AccountView
 Import(context.Context,string)(ojadapter.ImportResult,error)
 PrepareOnline(context.Context,ojadapter.Target,string)(ojadapter.SubmitPlan,error)
 SubmitOnline(context.Context,ojadapter.SubmitPlan,string,string)(ojadapter.Receipt,error)
 ReconcileOnline(context.Context,string)(ojadapter.Reconciliation,error)
 Judge(context.Context,ojadapter.Target,string)(ojadapter.JudgeStatus,error)
 Login(context.Context,string,string,bool)(ojadapter.AccountView,error)
 Restore(context.Context)(ojadapter.AccountView,error)
 Refresh(context.Context)(ojadapter.AccountView,error)
 Logout()(ojadapter.AccountView,error)
 Close()
}

type ojPending struct {
	root, token string
	plan        ojadapter.Plan
 result ojadapter.ImportResult
 ext string
}
type OJPreview struct {
	Token string         `json:"token"`
	Plan  ojadapter.Plan `json:"plan"`
}
type OJApplyResult struct {
	Nav       Navigation `json:"nav"`
	Completed []string   `json:"completed"`
	Skipped   []string   `json:"skipped"`
	Failed    []string   `json:"failed"`
}

func ojRevision(path string) (string, error) { s, e := localrun.LoadCases(path); return s.Revision, e }
func (d *Desktop) OJPreviewImport(link, ext string, attach bool) (OJPreview, error) {return d.ojPreviewImport(link,ext,attach,"")}
func(d *Desktop) OJPreviewImportAt(link,ext string,attach bool,dir string)(OJPreview,error){return d.ojPreviewImport(link,ext,attach,dir)}
func (d *Desktop) ojPreviewImport(link, ext string, attach bool, destination string) (OJPreview, error) {
	d.mu.Lock()
	d.ojPending = nil
	d.ojSequence++
	seq := d.ojSequence
	root, source, dir := d.root, d.selected.Path, d.root
	if source != "" {
		dir = d.selected.Directory
	}
	if !attach&&destination!=""{dir=destination}
 running := d.running
	d.mu.Unlock()
	if running || root == "" || attach && source == "" {
		return OJPreview{}, errors.New("请先选择文件夹或待关联文件")
	}
	if !attach {
		source = ""
	}
	if _, e := localrun.Within(root, dir); e != nil {
		return OJPreview{}, e
	}
	if d.accounts==nil||!d.accounts.Status().Available{return OJPreview{},errors.New("Adapter 不可用")}
 result, e := d.accounts.Import(context.Background(), link)
	if e != nil {
		return OJPreview{}, e
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.running || d.root != root || d.ojSequence != seq || attach && d.selected.Path != source {
		return OJPreview{}, errors.New("目录或当前文件已变化，请重新导入")
	}
	if _, e = localrun.Within(root, dir); e != nil {
		return OJPreview{}, e
	}
	if attach {
		if _, e = localrun.Within(root, source); e != nil {
			return OJPreview{}, e
		}
	}
	var p ojadapter.Plan
 archive:=false;for _,problem:=range result.Problems{archive=archive||problem.Statement!=""}
 if archive {p,e=ojadapter.BuildArchivePlan(result,dir,ext,source,func(path string)(string,bool,error){s,e:=localrun.LoadCases(path);return s.Revision,len(s.Cases)==0,e})}else{p,e=ojadapter.BuildPlan(result, dir, ext, source, ojRevision)}
	if e != nil {
		return OJPreview{}, e
	}
	token := fmt.Sprint(seq)
	d.ojPending = &ojPending{root:root,token:token,plan:p,result:result,ext:ext}
	return OJPreview{token, p}, nil
}
func (d *Desktop) OJApplyImport(token string) (OJApplyResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := OJApplyResult{Nav: d.nav(), Completed: []string{}, Skipped: []string{}, Failed: []string{}}
	pending := d.ojPending
	d.ojPending = nil
	if d.running || pending == nil || pending.token != token || pending.root != d.root || pending.plan.Attach != "" && pending.plan.Attach != d.selected.Path {
		return out, errors.New("导入预览已失效")
	}
	p := pending.plan
	if a := d.accounts.Status(); !a.Available || a.ID != p.AccountID || a.State != "signed_in" {
		return out, errors.New("账号状态已变化，请重新预览")
	}
	if _, e := localrun.Within(d.root, p.Directory); e != nil {
		return out, e
	}
	var first localrun.Source
 packageRoot:="";cleanup:=func(){};for _,row:=range p.Rows{if row.Action=="create"&&strings.EqualFold(filepath.Ext(row.Name),".c"){var err error;packageRoot,cleanup,err=d.installedKit();if err!=nil{packageRoot="";d.kitError=err.Error()}else{d.kitError=""};break}};defer cleanup()
	for _, row := range p.Rows {
		if row.Action == "skip" {
			out.Skipped = append(out.Skipped, row.Name)
			continue
		}
		path := filepath.Join(p.Directory, row.Name)
		if e := p.Recheck(row, ojRevision); e != nil {
			out.Failed = append(out.Failed, row.Name+"："+e.Error())
			continue
		}
		var s localrun.Source
		var e error
		if row.Action == "create" {
			dir:=filepath.Dir(path)
   if p.Archive{e=ensureImportDir(d.root,p.Directory,dir)}
   if e==nil{s, e = localrun.CreateSourceWithKit(d.root, dir, filepath.Base(path), packageRoot)}
		} else {
			if _, e = localrun.Within(d.root, path); e == nil {
				s, e = localrun.Open(path)
			}
		}
		if e != nil {
			out.Failed = append(out.Failed, row.Name+"："+e.Error())
			continue
		}
		if row.Samples > 0 && (p.Archive||row.Revision()=="missing") {
			cs := []localrun.Case{}
			for _, sample := range row.Problem().Samples {
				expected := sample.Expected
				cs = append(cs, localrun.Case{Input: sample.Input, Expected: &expected})
			}
			if _, e = localrun.SaveCases(s.Path, cs, row.Revision()); e != nil {
				out.Failed = append(out.Failed, row.Name+"：样例未保存；源码保留："+e.Error())
				continue
			}
		}
		if row.Action!="repair"{binding:=row.Problem().Binding();if binding.Target.Kind=="contest"{binding.ContestTitle=p.Title};e=ojadapter.SaveBinding(s.Path,binding)}
  if e != nil {
			out.Failed = append(out.Failed, row.Name+"：关联未保存；源码和已有样例保留："+e.Error())
			continue
		}
		if row.Document!=""{if e=localrun.SaveDocument(d.root,s.Path,row.Document,row.Problem().Markdown());e!=nil{out.Failed=append(out.Failed,row.Name+"：题面未保存；源码、关联及样例保留："+e.Error());continue}}
  out.Completed = append(out.Completed, row.Name)
		delete(d.last, s.Path)
		if first.Path == "" {
			first = s
		}
	}
	if first.Path != "" {
		d.selected = first
		d.prepared = nil
	}
	out.Nav = d.nav()
	return out, nil
}
func (d *Desktop) OJBinding() (*ojadapter.Binding, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.selected.Path == "" {
		return nil, nil
	}
	if _, e := localrun.Within(d.root, d.selected.Path); e != nil {
		return nil, e
	}
	return ojadapter.LoadBinding(d.selected.Path)
}
func (d *Desktop) OJUnbind(path string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.running || path == "" || path != d.selected.Path {
		return errors.New("当前文件已变化")
	}
	if _, e := localrun.Within(d.root, path); e != nil {
		return e
	}
	return ojadapter.RemoveBinding(path)
}
func (d *Desktop) OJOpenOriginal(path string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if path == "" || path != d.selected.Path {
		return errors.New("当前文件已变化")
	}
	if _, e := localrun.Within(d.root, path); e != nil {
		return e
	}
	b, e := ojadapter.LoadBinding(path)
	if e != nil {
		return e
	}
	if b == nil {
		return errors.New("当前文件未关联题目")
	}
	return localrun.OpenOJLink(b.SourceURL)
}

func ensureImportDir(root,base,target string)error{
 rel,e:=filepath.Rel(base,target);if e!=nil||rel==".."||strings.HasPrefix(rel,".."+string(filepath.Separator)){return errors.New("导入路径无效")};if rel=="."{_,e=localrun.Within(root,base);return e}
 current:=base
 for _,part:=range strings.Split(rel,string(filepath.Separator)){next:=filepath.Join(current,part);if _,e=os.Lstat(next);os.IsNotExist(e){if _,e=localrun.CreateFolder(root,current,part);e!=nil{return e}}else if e!=nil{return e};if _,e=localrun.Within(root,next);e!=nil{return e};current=next}
 return nil
}

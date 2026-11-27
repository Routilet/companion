package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/Routilet/companion/internal/localrun"
	"github.com/Routilet/companion/internal/ojadapter"
	"path/filepath"
)

type ojPending struct {
	root, token string
	plan        ojadapter.Plan
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
func (d *Desktop) OJPreviewImport(link, ext string, attach bool) (OJPreview, error) {
	d.mu.Lock()
	d.ojPending = nil
	d.ojSequence++
	seq := d.ojSequence
	root, source, dir := d.root, d.selected.Path, d.root
	if source != "" {
		dir = d.selected.Directory
	}
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
	p, e := ojadapter.BuildPlan(result, dir, ext, source, ojRevision)
	if e != nil {
		return OJPreview{}, e
	}
	token := fmt.Sprint(seq)
	d.ojPending = &ojPending{root, token, p}
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
	if a := d.accounts.Status(); a.ID != p.AccountID || a.State != "signed_in" {
		return out, errors.New("账号状态已变化，请重新预览")
	}
	if _, e := localrun.Within(d.root, p.Directory); e != nil {
		return out, e
	}
	var first localrun.Source
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
			s, e = localrun.CreateSourceWithKit(d.root, p.Directory, row.Name, d.kitPackage)
		} else {
			if _, e = localrun.Within(d.root, path); e == nil {
				s, e = localrun.Open(path)
			}
		}
		if e != nil {
			out.Failed = append(out.Failed, row.Name+"："+e.Error())
			continue
		}
		if row.Revision() == "missing" && row.Samples > 0 {
			cs := []localrun.Case{}
			for _, sample := range row.Problem().Samples {
				expected := sample.Expected
				cs = append(cs, localrun.Case{Input: sample.Input, Expected: &expected})
			}
			if _, e = localrun.SaveCases(s.Path, cs, "missing"); e != nil {
				out.Failed = append(out.Failed, row.Name+"：样例未保存；源码保留："+e.Error())
				continue
			}
		}
		if e = ojadapter.SaveBinding(s.Path, row.Problem().Binding()); e != nil {
			out.Failed = append(out.Failed, row.Name+"：关联未保存；源码和已有样例保留："+e.Error())
			continue
		}
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

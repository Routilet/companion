package main

import (
	"context"
	"errors"
	"github.com/Routilet/companion/internal/localrun"
	"github.com/Routilet/companion/internal/ojadapter"
)

type ojSubmissionPending struct {
	preview  *localrun.Submission
	binding  ojadapter.Binding
	compiler string
	plan     ojadapter.SubmitPlan
}
type OJSubmissionPreview struct {
	Plan     ojadapter.SubmitPlan `json:"plan"`
	Title    string               `json:"title"`
	Label    string               `json:"label"`
	Nickname string               `json:"nickname"`
}

func (d *Desktop) OJSubmissionAvailable() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.selected.Path == "" || d.running || d.accounts == nil {
		return false
	}
	if _, e := localrun.Within(d.root, d.selected.Path); e != nil {
		return false
	}
	b, e := ojadapter.LoadBinding(d.selected.Path)
	return e == nil && b != nil && d.accounts.Status().Available
}
func (d *Desktop) checkOJPreview(compiler, digest string) (*localrun.Submission, *ojadapter.Binding, error) {
	p := d.prepared
	if d.running || p == nil || p.Source.Path != d.selected.Path || p.Digest != digest {
		return nil, nil, errors.New("源码预览已失效，请重新预览")
	}
	if _, e := localrun.Within(d.root, p.Source.Path); e != nil {
		return nil, nil, e
	}
	if _, e := localrun.CheckSubmission(*p, compiler); e != nil {
		return nil, nil, e
	}
	b, e := ojadapter.LoadBinding(p.Source.Path)
	if e != nil {
		return nil, nil, e
	}
	if b == nil {
		return nil, nil, errors.New("当前文件未关联题目")
	}
	return p, b, nil
}
func (d *Desktop) OJPrepareSubmission(compiler, digest string) (OJSubmissionPreview, error) {
	d.mu.Lock()
	d.ojSubmit = nil
	p, b, e := d.checkOJPreview(compiler, digest)
	d.mu.Unlock()
	if e != nil {
		return OJSubmissionPreview{}, e
	}
	plan, e := d.accounts.PrepareOnline(context.Background(), b.Target, p.Text)
	if e != nil {
		return OJSubmissionPreview{}, e
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	current, binding, e := d.checkOJPreview(compiler, digest)
	if e != nil {
		return OJSubmissionPreview{}, e
	}
	view := d.accounts.Status()
	if current != p || *binding != *b || plan.Target != b.Target || plan.SourceHash != ojadapter.SourceHash(p.Text) || plan.AccountID != view.ID || view.State != "signed_in" {
		return OJSubmissionPreview{}, errors.New("源码、关联或账号已变化，请重新预览")
	}
	d.ojSubmit = &ojSubmissionPending{p, *b, compiler, plan}
	return OJSubmissionPreview{plan, b.Title, b.Label, view.Nickname}, nil
}
func (d *Desktop) OJSendSubmission(compiler, digest, token, language string) (ojadapter.Receipt, error) {
	d.mu.Lock()
	pending := d.ojSubmit
	d.ojSubmit = nil
	if pending == nil || pending.plan.Token != token || pending.compiler != compiler || !pending.plan.Allows(language) {
		d.mu.Unlock()
		return ojadapter.Receipt{}, errors.New("提交确认已失效，请重新预览")
	}
	p, b, e := d.checkOJPreview(compiler, digest)
	if e != nil {
		d.mu.Unlock()
		return ojadapter.Receipt{}, e
	}
	view := d.accounts.Status()
	if p != pending.preview || *b != pending.binding || view.ID != pending.plan.AccountID || view.State != "signed_in" {
		d.mu.Unlock()
		return ojadapter.Receipt{}, errors.New("源码、关联或账号已变化，请重新预览")
	}
	code := p.Text
	d.mu.Unlock()
	// Send only the exact preview bytes, never reread from the UI or regenerate after confirmation.
	return d.accounts.SubmitOnline(context.Background(), pending.plan, language, code)
}

func (d *Desktop) OJReconcileSubmission(compiler, digest string) (ojadapter.Reconciliation, error) {
	d.mu.Lock()
	d.ojSubmit = nil
	_, b, e := d.checkOJPreview(compiler, digest)
	d.mu.Unlock()
	if e != nil {
		return ojadapter.Reconciliation{}, e
	}
	return d.accounts.ReconcileOnline(context.Background(), b.Target.ProblemID)
}

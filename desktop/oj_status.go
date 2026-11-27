package main

import (
	"context"
	"errors"
	"github.com/Routilet/companion/internal/localrun"
	"github.com/Routilet/companion/internal/ojadapter"
	"time"
)

type judgeKey struct {
	Account string
	Target  ojadapter.Target
}
type judgeView struct {
	Status  ojadapter.JudgeStatus `json:"status"`
	Checked string                `json:"checked"`
}

// All cache accesses use Desktop.mu; never hold it during a network call.
func (d *Desktop) rememberJudge(s ojadapter.JudgeStatus) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.ojJudges == nil || len(d.ojJudges) >= 512 {
		d.ojJudges = map[judgeKey]judgeView{}
	}
	k := judgeKey{s.AccountID, s.Target}
	old := d.ojJudges[k]
	d.ojJudges[k] = judgeView{ojadapter.MergeJudge(old.Status, s), time.Now().Format(time.RFC3339)}
}
func (d *Desktop) rememberReceipt(r ojadapter.Receipt) {
	d.rememberJudge(ojadapter.JudgeStatus{AccountID: r.AccountID, Target: r.Target, Accepted: r.Result == "AC", Latest: &ojadapter.JudgeResult{ID: r.ID, Result: r.Result, Language: r.Language}})
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ojReceipt = &r
}
func (d *Desktop) judgeBinding(root, path string) (*ojadapter.Binding, error) {
	if path == "" {
		return nil, nil
	}
	p, e := localrun.Within(root, path)
	if e != nil {
		return nil, e
	}
	return ojadapter.LoadBinding(p)
}

// Only local data is read; no startup, scan or selection contacts OJ.
func (d *Desktop) OJCachedJudges(paths []string) map[string]judgeView {
	out := map[string]judgeView{}
	if d.accounts == nil || len(paths) > 512 {
		return out
	}
	a := d.accounts.Status()
	if a.ID == "" || a.State == "signed_out" {
		return out
	}
	d.mu.Lock()
	root := d.root
	defer d.mu.Unlock()
	for _, p := range paths {
		b, e := d.judgeBinding(root, p)
		if e != nil || b == nil {
			continue
		}
		if v, ok := d.ojJudges[judgeKey{a.ID, b.Target}]; ok {
			out[p] = v
		}
	}
	return out
}

// Explicit refresh only. Limit a batch to 20 distinct remote targets; errors
// stop the batch and retain earlier results without fabricating a verdict.
func (d *Desktop) OJRefreshJudges(paths []string) (int, error) {
	if d.accounts == nil || len(paths) > 512 {
		return 0, errors.New("无效刷新范围")
	}
	a := d.accounts.Status()
	if a.ID == "" || a.State == "signed_out" {
		return 0, errors.New("请先登录 Accoding")
	}
	d.mu.Lock()
	root := d.root
	d.mu.Unlock()
	targets := []ojadapter.Target{}
	seen := map[ojadapter.Target]bool{}
	for _, p := range paths {
		b, e := d.judgeBinding(root, p)
		if e != nil || b == nil || seen[b.Target] {
			continue
		}
		seen[b.Target] = true
		targets = append(targets, b.Target)
	}
	if len(targets) > 20 {
		return 0, errors.New("一次最多刷新 20 道题；请收起其他目录或单独刷新当前题目")
	}
	count := 0
	batch, stop := context.WithTimeout(context.Background(), 60*time.Second)
	defer stop()
	for _, t := range targets {
		if batch.Err() != nil {
			return count, errors.New("刷新已超时；保留已有结果")
		}
		if d.accounts.Status().ID != a.ID {
			return count, errors.New("账号已变化，停止刷新")
		}
		ctx, cancel := context.WithTimeout(batch, 15*time.Second)
		v, e := d.accounts.Judge(ctx, t, "")
		cancel()
		if e != nil {
			return count, e
		}
		if v.AccountID != a.ID {
			return count, errors.New("账号已变化，停止刷新")
		}
		d.rememberJudge(v)
		count++
	}
	return count, nil
}

// Only poll the most recent confirmed receipt from this process. Never submit
// or poll a different account, never start or restore an adapter session.
func (d *Desktop) OJPollReceipt(id string) (ojadapter.JudgeResult, error) {
	d.mu.Lock()
	r := d.ojReceipt
	d.mu.Unlock()
	if r == nil || r.ID != id || d.accounts == nil || d.accounts.Status().ID != r.AccountID {
		return ojadapter.JudgeResult{}, errors.New("提交或账号已变化，停止查询")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	v, e := d.accounts.Judge(ctx, r.Target, id)
	if e != nil {
		return ojadapter.JudgeResult{}, e
	}
	if v.AccountID != r.AccountID || v.Latest == nil {
		return ojadapter.JudgeResult{}, errors.New("评测回执不匹配")
	}
	d.rememberJudge(v)
	return *v.Latest, nil
}

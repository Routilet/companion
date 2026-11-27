package ojadapter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

type SubmitPlan struct {
	Token      string   `json:"token"`
	Target     Target   `json:"target"`
	AccountID  string   `json:"accountId"`
	Languages  []string `json:"languages"`
	SourceHash string   `json:"sourceHash"`
}
type Receipt struct {
	ID         string `json:"id"`
	Target     Target `json:"target"`
	AccountID  string `json:"accountId"`
	Language   string `json:"language"`
	Result     string `json:"result"`
	SourceHash string `json:"sourceHash"`
}

func SourceHash(code string) string {
	h := sha256.Sum256([]byte(code))
	return hex.EncodeToString(h[:])
}
func validToken(s string) bool {
	b, e := hex.DecodeString(s)
	return e == nil && len(b) == 16 && s == strings.ToLower(s)
}
func (p SubmitPlan) valid(target Target, id, hash string) bool {
	if !validToken(p.Token) || p.Target != target || p.AccountID != id || p.SourceHash != hash || len(p.Languages) == 0 || len(p.Languages) > 128 {
		return false
	}
	seen := map[string]bool{}
	for _, l := range p.Languages {
		if l == "" || !validText(l, 128) || seen[l] {
			return false
		}
		seen[l] = true
	}
	return true
}
func (p SubmitPlan) Allows(lang string) bool {
	for _, l := range p.Languages {
		if l == lang {
			return true
		}
	}
	return false
}
func (a *Accounts) PrepareOnline(ctx context.Context, t Target, code string) (SubmitPlan, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	ctx, done := a.context(ctx)
	defer done()
	var p SubmitPlan
	if a.closed || a.session == nil || a.state.ID == "" {
		return p, errors.New("请先登录 Accoding")
	}
	if t.URL() == "" || code == "" || !validText(code, 4<<20) {
		return p, errors.New("提交源码或目标无效")
	}
	hash := SourceHash(code)
	id := a.state.ID
	if e := a.session.Call(ctx, "prepare", map[string]any{"target": t, "sourceHash": hash}, &p); e != nil {
		return SubmitPlan{}, a.submitError(e, false)
	}
	if !p.valid(t, id, hash) {
		a.reset()
		return SubmitPlan{}, errors.New("远程提交预览不匹配")
	}
	a.verified = true
	return p, nil
}
func (a *Accounts) SubmitOnline(ctx context.Context, p SubmitPlan, language, code string) (Receipt, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	ctx, done := a.context(ctx)
	defer done()
	var r Receipt
	if a.closed || a.session == nil || a.state.ID == "" || !p.valid(p.Target, a.state.ID, SourceHash(code)) || p.Target.URL() == "" || !p.Allows(language) || code == "" || !validText(code, 4<<20) {
		return r, errors.New("提交确认已失效，请重新预览")
	}
	if e := a.session.Call(ctx, "submit", map[string]string{"token": p.Token, "language": language, "code": code}, &r); e != nil {
		return Receipt{}, a.submitError(e, true)
	}
	if !numberID.MatchString(r.ID) || r.AccountID != p.AccountID || r.Target != p.Target || r.Language != language || r.SourceHash != p.SourceHash || r.Result == "" || !validText(r.Result, 128) {
		a.reset()
		return Receipt{}, errors.New("提交回执无效，结果可能不确定；请在 OJ 核对，勿重发")
	}
	return r, nil
}
func (a *Accounts) submitError(e error, sent bool) error {
	var remote *RemoteError
	if !errors.As(e, &remote) {
		a.reset()
		if sent {
			return errors.New("提交通信中断，可能已发送；请在 OJ 核对，勿重发")
		}
		return errors.New("无法获取提交目标，请重新登录")
	}
	switch remote.Code {
	case "auth":
		a.reset()
		return errors.New("账号已变化或登录失效，请重新登录")
	case "unknown_outcome":
		return errors.New("存在未决提交或结果不确定，请在 OJ 核对；禁止直接重发")
	case "forbidden":
		return errors.New("目标无权提交或比赛未开始/已结束；不会自动切换目标")
	case "confirmation_expired":
		return errors.New("提交确认已失效，请重新预览")
	case "submission_unavailable":
		return errors.New("提交保护记录不可用，未发送")
	case "invalid_params":
		return errors.New("提交参数无效，请重新预览")
	default:
		if sent {
			return errors.New("提交未获有效回执；请在 OJ 核对，勿直接重发")
		}
		return errors.New("无法核验远程目标或语言，请稍后重试")
	}
}

type Reconciliation struct {
	State  string `json:"state"`
	ID     string `json:"id,omitempty"`
	Result string `json:"result,omitempty"`
}

func (a *Accounts) ReconcileOnline(ctx context.Context, problem string) (Reconciliation, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	ctx, done := a.context(ctx)
	defer done()
	var r Reconciliation
	if a.closed || a.session == nil || a.state.ID == "" || !numberID.MatchString(problem) {
		return r, errors.New("请先登录并选择关联题目")
	}
	if e := a.session.Call(ctx, "reconcile", map[string]string{"problemId": problem}, &r); e != nil {
		return Reconciliation{}, a.submitError(e, false)
	}
	switch r.State {
	case "none", "unresolved", "manual_required":
		if r.ID != "" || r.Result != "" {
			return Reconciliation{}, ErrProtocol
		}
	case "resolved":
		if !numberID.MatchString(r.ID) || r.Result == "" || !validText(r.Result, 128) {
			return Reconciliation{}, ErrProtocol
		}
	default:
		return Reconciliation{}, ErrProtocol
	}
	return r, nil
}

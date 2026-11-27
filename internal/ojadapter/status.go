package ojadapter

import (
	"context"
	"errors"
)

type JudgeResult struct {
	ID       string `json:"id"`
	Result   string `json:"result"`
	Language string `json:"language"`
}
type JudgeStatus struct {
	AccountID string       `json:"accountId"`
	Target    Target       `json:"target"`
	Accepted  bool         `json:"accepted"`
	Latest    *JudgeResult `json:"latest,omitempty"`
}

func (s JudgeStatus) valid(t Target, account, id string) bool {
	if s.Target != t || s.AccountID != account || t.URL() == "" || !numberID.MatchString(account) {
		return false
	}
	if s.Latest == nil {
		return id == "" && !s.Accepted
	}
	r := s.Latest
	return numberID.MatchString(r.ID) && (id == "" || r.ID == id) && r.Result != "" && validText(r.Result, 128) && validText(r.Language, 128) && (r.Result != "AC" || s.Accepted) && (id == "" || s.Accepted == (r.Result == "AC"))
}
func (a *Accounts) Judge(ctx context.Context, t Target, id string) (JudgeStatus, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	ctx, done := a.context(ctx)
	defer done()
	var s JudgeStatus
	if a.closed || a.session == nil || a.state.ID == "" || t.URL() == "" || id != "" && !numberID.MatchString(id) {
		return s, errors.New("请先登录并选择关联题目")
	}
	method := "status"
	p := map[string]any{"target": t}
	if id != "" {
		method = "result"
		p["id"] = id
	}
	if e := a.session.Call(ctx, method, p, &s); e != nil {
		return JudgeStatus{}, a.submitError(e, false)
	}
	if !s.valid(t, a.state.ID, id) {
		a.reset()
		return JudgeStatus{}, ErrProtocol
	}
	a.verified = true
	return s, nil
}
func NewerID(a, b string) bool { return len(a) > len(b) || len(a) == len(b) && a > b }

// MergeJudge preserves evidence of an observed AC separately from the newest visible submission.
func MergeJudge(old, next JudgeStatus) JudgeStatus {
	if old.AccountID != next.AccountID || old.Target != next.Target {
		return next
	}
	next.Accepted = next.Accepted || old.Accepted
	if old.Latest != nil && (next.Latest == nil || NewerID(old.Latest.ID, next.Latest.ID)) {
		next.Latest = old.Latest
	}
	return next
}

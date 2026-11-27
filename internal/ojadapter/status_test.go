package ojadapter

import (
	"context"
	"testing"
)

func TestMergeJudge(t *testing.T) {
	target := Target{Kind: "contest", ContestID: "42", ProblemID: "5"}
	a := JudgeStatus{AccountID: "7", Target: target, Accepted: true, Latest: &JudgeResult{ID: "99", Result: "AC"}}
	b := JudgeStatus{AccountID: "7", Target: target, Latest: &JudgeResult{ID: "100", Result: "WA"}}
	s := MergeJudge(a, b)
	if !s.Accepted || s.Latest.ID != "100" {
		t.Fatal(s)
	}
	if MergeJudge(s, a).Latest.ID != "100" {
		t.Fatal("older result replaced latest")
	}
	b.AccountID = "8"
	if MergeJudge(a, b).Accepted {
		t.Fatal("cross-account AC")
	}
	if a.valid(target, "8", "") || a.valid(target, "7", "100") {
		t.Fatal("identity accepted")
	}
}
func TestJudgeSignedOut(t *testing.T) {
	a := NewAccounts("")
	defer a.Close()
	if _, e := a.Judge(context.Background(), Target{Kind: "contest", ContestID: "42", ProblemID: "5"}, ""); e == nil {
		t.Fatal("signed out query")
	}
}
func TestJudgeFixture(t *testing.T) {
	a := NewAccounts(accountFixture(t))
	a.store = &memoryStore{}
	defer a.Close()
	ctx := context.Background()
	if _, e := a.Login(ctx, "7", "test-secret", false); e != nil {
		t.Fatal(e)
	}
	target := Target{Kind: "contest", ContestID: "42", ProblemID: "5"}
	s, e := a.Judge(ctx, target, "")
	if e != nil || !s.Accepted || s.Latest == nil || s.Latest.Result != "WA" {
		t.Fatal(s, e)
	}
	s, e = a.Judge(ctx, target, "123")
	if e != nil || s.Latest == nil || s.Latest.ID != "123" || s.Latest.Result != "AC" {
		t.Fatal(s, e)
	}
}

package ojadapter

import (
	"context"
	"strings"
	"testing"
)

func TestOnlineSubmission(t *testing.T) {
	a := NewAccounts(accountFixture(t))
	a.store = &memoryStore{}
	defer a.Close()
	ctx := context.Background()
	target := Target{Kind: "contest", ContestID: "42", ProblemID: "5"}
	code := "int main(void){}"
	if _, e := a.PrepareOnline(ctx, target, code); e == nil {
		t.Fatal("no auth")
	}
	if _, e := a.Login(ctx, "7", "test-secret", false); e != nil {
		t.Fatal(e)
	}
	p, e := a.PrepareOnline(ctx, target, code)
	if e != nil || p.AccountID != "7" {
		t.Fatal(p, e)
	}
	if _, e = a.SubmitOnline(ctx, p, "C99", code+" "); e == nil {
		t.Fatal("changed source")
	}
	if _, e = a.SubmitOnline(ctx, p, "cpp", code); e == nil {
		t.Fatal("wrong language")
	}
	r, e := a.SubmitOnline(ctx, p, "C99", code)
	if e != nil || r.ID != "123" || r.Result != "WT" {
		t.Fatal(r, e)
	}
	if _, e = a.SubmitOnline(ctx, p, "C99", code); e == nil {
		t.Fatal("replay")
	}
	if _, e = a.Login(ctx, "66", "test-secret", false); e != nil {
		t.Fatal(e)
	}
	p, e = a.PrepareOnline(ctx, target, code)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = a.SubmitOnline(ctx, p, "C99", code); e == nil || !strings.Contains(e.Error(), "勿重发") {
		t.Fatal(e)
	}
	if a.Status().State != "signed_out" {
		t.Fatal("stale session")
	}
}

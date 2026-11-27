package ojadapter

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func problem() Problem {
	t := Target{Kind: "contest", ContestID: "42", ProblemID: "5"}
	return Problem{Target: t, Title: "题目", Label: "A", SourceURL: t.URL(), Samples: []Sample{{"", ""}}}
}
func TestImportValidation(t *testing.T) {
	p := problem()
	r := ImportResult{"7", "作业", []Problem{p}}
	if e := r.Validate("7"); e != nil {
		t.Fatal(e)
	}
	if e := r.Validate("8"); e == nil {
		t.Fatal("wrong account")
	}
	r.Problems = append(r.Problems, p)
	if e := r.Validate("7"); e == nil {
		t.Fatal("duplicate")
	}
	r.Problems = []Problem{p}
	r.Problems[0].SourceURL = "https://evil.test"
	if e := r.Validate("7"); e == nil {
		t.Fatal("foreign url")
	}
}
func TestBindingLifecycle(t *testing.T) {
	d := t.TempDir()
	a := filepath.Join(d, "A.c")
	b := filepath.Join(d, "B.c")
	os.WriteFile(a, []byte("code"), 0600)
	p := problem().Binding()
	if e := SaveBinding(a, p); e != nil {
		t.Fatal(e)
	}
	if e := SaveBinding(a, p); e == nil {
		t.Fatal("overwrite")
	}
	if e := RenameBound(a, b, func() error { return errors.New("failed") }); e == nil {
		t.Fatal("rename")
	}
	if v, _ := LoadBinding(b); v != nil {
		t.Fatal("rollback")
	}
	if e := RenameBound(a, b, func() error { return os.Rename(a, b) }); e != nil {
		t.Fatal(e)
	}
	v, e := LoadBinding(b)
	if e != nil || v == nil || v.Target != p.Target {
		t.Fatal(v, e)
	}
	if v, _ := LoadBinding(a); v != nil {
		t.Fatal("old binding")
	}
	if e := RemoveBinding(b); e != nil {
		t.Fatal(e)
	}
	if raw, _ := os.ReadFile(b); string(raw) != "code" {
		t.Fatal("source modified")
	}
}
func TestOrphanAndSymlink(t *testing.T) {
	d := t.TempDir()
	a := filepath.Join(d, "A.c")
	if e := SaveBinding(a, problem().Binding()); e != nil {
		t.Fatal(e)
	}
	if b, e := LoadBinding(a); e != nil || b == nil {
		t.Fatal(b, e)
	}
	x := t.TempDir()
	if e := os.Symlink(d, filepath.Join(x, ".companion")); e != nil {
		t.Skip(e)
	}
	if e := SaveBinding(filepath.Join(x, "A.c"), problem().Binding()); e == nil {
		t.Fatal("symlink accepted")
	}
}
func TestPlanConflictAndAttach(t *testing.T) {
	d := t.TempDir()
	p := problem()
	r := ImportResult{"7", "作业", []Problem{p}}
	rev := func(string) (string, error) { return "existing", nil }
	a := filepath.Join(d, "A.c")
	os.WriteFile(a, []byte("keep"), 0600)
	plan, e := BuildPlan(r, d, ".c", "", rev)
	if e != nil || plan.Rows[0].Action != "skip" {
		t.Fatal(plan, e)
	}
	plan, e = BuildPlan(r, d, ".c", a, rev)
	if e != nil || plan.Rows[0].Action != "attach" || plan.Rows[0].Samples != 0 {
		t.Fatal(plan, e)
	}
	if e = plan.Recheck(plan.Rows[0], rev); e != nil {
		t.Fatal(e)
	}
	if e = plan.Recheck(plan.Rows[0], func(string) (string, error) { return "changed", nil }); e == nil {
		t.Fatal("stale revision")
	}
	if e = SaveBinding(a, p.Binding()); e != nil {
		t.Fatal(e)
	}
	if e = plan.Recheck(plan.Rows[0], rev); e == nil {
		t.Fatal("changed binding")
	}
	os.Remove(a)
	plan, e = BuildPlan(r, d, ".c", "", rev)
	if e != nil || plan.Rows[0].Action != "skip" {
		t.Fatal("orphan", plan, e)
	}
}
func TestPlanRecheckNew(t *testing.T) {
	d := t.TempDir()
	p := problem()
	r := ImportResult{"7", "作业", []Problem{p}}
	rev := func(string) (string, error) { return "missing", nil }
	plan, e := BuildPlan(r, d, ".c", "", rev)
	if e != nil || plan.Rows[0].Action != "create" {
		t.Fatal(plan, e)
	}
	os.WriteFile(filepath.Join(d, "A.c"), []byte("keep"), 0600)
	if e = plan.Recheck(plan.Rows[0], rev); e == nil {
		t.Fatal("overwrites existing")
	}
}

func TestImportSession(t *testing.T) {
	a := NewAccounts(accountFixture(t))
	a.store = &memoryStore{}
	defer a.Close()
	ctx := context.Background()
	if _, e := a.Import(ctx, "x"); e == nil || a.session != nil {
		t.Fatal("unauthenticated import")
	}
	if _, e := a.Login(ctx, "42", "test-secret", false); e != nil {
		t.Fatal(e)
	}
	r, e := a.Import(ctx, "https://accoding.buaa.edu.cn/contest-ng/index.html#/42")
	if e != nil || len(r.Problems) != 1 || r.AccountID != "42" {
		t.Fatal(r, e)
	}
}

package ojadapter

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

type memoryStore struct {
	c    credentials
	fail bool
}

func (*memoryStore) Supported() bool { return true }
func (s *memoryStore) Load() (credentials, error) {
	if s.c.Username == "" {
		return credentials{}, errNoCredential
	}
	return s.c, nil
}
func (s *memoryStore) Save(c credentials) error {
	if s.fail {
		return errCredentialStore
	}
	s.c = c
	return nil
}
func (s *memoryStore) Delete() error { s.c = credentials{}; return nil }
func accountFixture(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "account")
	if runtime.GOOS == "windows" {
		p += ".exe"
	}
	if b, e := exec.Command("go", "build", "-o", p, "./testdata/accountfixture").CombinedOutput(); e != nil {
		t.Fatalf("%v %s", e, b)
	}
	return p
}
func TestAccounts(t *testing.T) {
	a := NewAccounts(accountFixture(t))
	m := &memoryStore{}
	a.store = m
	defer a.Close()
	ctx := context.Background()
	if s := a.Status(); !s.Available || s.State != "signed_out" || a.session != nil {
		t.Fatal(s)
	}
	v, e := a.Login(ctx, "42", "test-secret", true)
	if e != nil || v.ID != "42" || !v.Saved {
		t.Fatal(v, e)
	}
	a.reset()
	v, e = a.Restore(ctx)
	if e != nil || v.ID != "42" {
		t.Fatal(v, e)
	}
	v, e = a.Login(ctx, "43", "wrong", false)
	if e == nil || v.State != "signed_out" || m.c.Username != "" {
		t.Fatal(v, e)
	}
	v, e = a.Login(ctx, "99", "test-secret", false)
	if e != nil {
		t.Fatal(e)
	}
	v, e = a.Refresh(ctx)
	if e == nil || v.State != "unverified" || v.ID != "99" {
		t.Fatal(v, e)
	}
	v, e = a.Login(ctx, "88", "test-secret", false)
	if e != nil {
		t.Fatal(e)
	}
	v, e = a.Refresh(ctx)
	if e == nil || v.State != "signed_out" {
		t.Fatal(v, e)
	}
	v, e = a.Logout()
	if e != nil || v.State != "signed_out" || a.session != nil {
		t.Fatal(v, e)
	}
}
func TestMissingAndCredentialCodec(t *testing.T) {
	a := NewAccounts(filepath.Join(t.TempDir(), "absent"))
	a.store = unavailableStore{}
	defer a.Close()
	if a.Status().Available {
		t.Fatal("missing available")
	}
	if _, e := a.Login(context.Background(), "1", "p", false); e == nil {
		t.Fatal("missing accepted")
	}
	b, e := encodeCredential(credentials{"用户名", "秘密"})
	if e != nil {
		t.Fatal(e)
	}
	c, e := decodeCredential(b)
	if e != nil || c.Password != "秘密" {
		t.Fatal(e)
	}
	if _, e = decodeCredential([]byte("{}")); !errors.Is(e, errCredentialStore) {
		t.Fatal(e)
	}
}

func TestSaveFailureAndClosed(t *testing.T) {
	a := NewAccounts(accountFixture(t))
	a.store = &memoryStore{fail: true}
	v, e := a.Login(context.Background(), "42", "test-secret", true)
	if e != nil || v.State != "signed_in" || v.Warning == "" || v.Saved {
		t.Fatal(v, e)
	}
	a.Close()
	if _, e = a.Restore(context.Background()); e == nil {
		t.Fatal("closed restore succeeded")
	}
	if a.Status().Available {
		t.Fatal("closed still available")
	}
}

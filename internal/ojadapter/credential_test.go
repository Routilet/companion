package ojadapter

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"testing"
)

// Uses a random, separate service slot, never the real user's saved credential.
func TestSystemCredentialRoundTrip(t *testing.T) {
	if os.Getenv("COMPANION_TEST_SYSTEM_CREDENTIALS") != "1" {
		t.Skip("opt-in native credential test")
	}
	var suffix [16]byte
	if _, e := rand.Read(suffix[:]); e != nil {
		t.Fatal(e)
	}
	s := systemCredentialStoreFor(credentialService + ".test." + hex.EncodeToString(suffix[:]))
	if !s.Supported() {
		t.Skip("native store unavailable")
	}
	t.Cleanup(func() {
		if e := s.Delete(); e != nil {
			t.Error("fixture cleanup failed")
		}
	})
	c := credentials{"测试账号", "fixture-only-secret"}
	if e := s.Save(c); e != nil {
		t.Fatal(e)
	}
	got, e := s.Load()
	if e != nil || got != c {
		t.Fatal("credential readback failed")
	}
	c.Password = "fixture-updated"
	if e = s.Save(c); e != nil {
		t.Fatal(e)
	}
	got, e = s.Load()
	if e != nil || got != c {
		t.Fatal("credential replacement failed")
	}
	if e = s.Delete(); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Load(); !errors.Is(e, errNoCredential) {
		t.Fatal("credential not removed")
	}
}

package ojadapter

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func fixture(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "fixture")
	if runtime.GOOS == "windows" {
		p += ".exe"
	}
	c := exec.Command("go", "build", "-o", p, "./testdata/fixture")
	if b, e := c.CombinedOutput(); e != nil {
		t.Fatalf("build fixture: %v %s", e, b)
	}
	return p
}
func start(t *testing.T, p string) *Session {
	t.Helper()
	s, e := Start(context.Background(), p, "fixture")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(s.Close)
	return s
}
func TestSession(t *testing.T) {
	p := fixture(t)
	s := start(t, p)
	if s.Capabilities().Protocol != Protocol {
		t.Fatal("handshake")
	}
	var out map[string]string
	if e := s.Call(context.Background(), "echo", map[string]string{"x": "中文\n"}, &out); e != nil || out["x"] != "中文\n" {
		t.Fatalf("echo: %v %v", out, e)
	}
	var re *RemoteError
	if e := s.Call(context.Background(), "error", nil, nil); !errors.As(e, &re) || re.Code != "fixture_error" {
		t.Fatal(e)
	}
	if e := s.Call(context.Background(), "echo", map[string]string{"x": "ok"}, &out); e != nil || out["x"] != "ok" {
		t.Fatal(e)
	}
	s.Close()
	s.Close()
	if e := s.Call(context.Background(), "echo", nil, nil); !errors.Is(e, ErrClosed) {
		t.Fatal(e)
	}
}
func TestFailures(t *testing.T) {
	p := fixture(t)
	for _, method := range []string{"bad-id", "both", "partial", "large", "crash", "invalid"} {
		t.Run(method, func(t *testing.T) {
			s := start(t, p)
			if e := s.Call(context.Background(), method, nil, nil); e == nil {
				t.Fatal("accepted failure")
			}
			if e := s.Call(context.Background(), "echo", nil, nil); !errors.Is(e, ErrClosed) {
				t.Fatal(e)
			}
		})
	}
}
func TestTimeoutAndClose(t *testing.T) {
	p := fixture(t)
	s := start(t, p)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if e := s.Call(ctx, "hang", nil, nil); !errors.Is(e, context.DeadlineExceeded) {
		t.Fatal(e)
	}
	if e := s.Call(context.Background(), "echo", nil, nil); !errors.Is(e, ErrClosed) {
		t.Fatal(e)
	}
	s = start(t, p)
	done := make(chan error, 1)
	go func() { done <- s.Call(context.Background(), "hang", nil, nil) }()
	s.Close()
	select {
	case e := <-done:
		if e == nil {
			t.Fatal("call succeeded")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("close blocked")
	}
}
func TestMissingAndWrongAdapter(t *testing.T) {
	if s, e := Start(context.Background(), "relative", "fixture"); e == nil {
		s.Close()
		t.Fatal("accepted relative")
	}
	if s, e := Start(context.Background(), filepath.Join(t.TempDir(), "missing"), "fixture"); e == nil {
		s.Close()
		t.Fatal("accepted missing")
	}
	if s, e := Start(context.Background(), fixture(t), "other"); e == nil {
		s.Close()
		t.Fatal("accepted wrong adapter")
	}
}
func TestRealAdapter(t *testing.T) {
	p := os.Getenv("COMPANION_TEST_ADAPTER")
	if p == "" {
		t.Skip("real binary not supplied")
	}
	s, e := Start(context.Background(), p, "accoding")
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	caps:=s.Capabilities()
 methods:=map[string]bool{}
 for _,m:=range caps.Methods{methods[m]=true}
 for _,m:=range []string{"capabilities","account","login","logout","resolve","import","prepare","submit","reconcile","result","status","import_document","contests","status_detail","result_detail","contest_status"}{if !methods[m]{t.Fatal("missing",m,caps)}}
 if caps.Version!="0.1.0"{t.Fatalf("adapter version: got %q, want 0.1.0",caps.Version)}
 for _,q:=range []struct{method string;params any}{
 {"import_document",map[string]string{"url":"https://accoding.buaa.edu.cn/problem/42/index"}},
 {"contests",struct{}{}},
 {"contest_status",map[string]any{"contestId":"42","problemIds":[]string{"5"}}},
 {"status_detail",map[string]any{"target":map[string]string{"kind":"contest","contestId":"42","problemId":"5"}}},
 {"result_detail",map[string]any{"target":map[string]string{"kind":"contest","contestId":"42","problemId":"5"},"id":"123"}},
 }{var re *RemoteError;if e=s.Call(context.Background(),q.method,q.params,nil);!errors.As(e,&re)||re.Code!="auth"{t.Fatal(q.method,e)}}
	var link struct {
		Kind    string `json:"kind"`
		ID      string `json:"id"`
		Service string `json:"service,omitempty"`
	}
	if e = s.Call(context.Background(), "resolve", map[string]string{"url": "https://accoding.buaa.edu.cn/problem/42/index"}, &link); e != nil || link.ID != "42" || link.Kind != "problemset" {
		t.Fatal(link, e)
	}
	var re *RemoteError
	if e = s.Call(context.Background(), "import", map[string]string{"url": "https://accoding.buaa.edu.cn/problem/42/index"}, nil); !errors.As(e, &re) || re.Code != "auth" {
		t.Fatal(e)
	}
	if e = s.Call(context.Background(), "submit", nil, nil); !errors.As(e, &re) || re.Code != "invalid_params" {
		t.Fatal(e)
	}
}

func TestConcurrentAndLocalRejection(t *testing.T) {
	s := start(t, fixture(t))
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var out map[string]string
			if e := s.Call(context.Background(), "echo", map[string]string{"x": "ok"}, &out); e != nil || out["x"] != "ok" {
				t.Errorf("echo: %v %v", out, e)
			}
		}()
	}
	wg.Wait()
	if e := s.Call(context.Background(), "echo", []string{"invalid"}, nil); e == nil {
		t.Fatal("accepted non-object params")
	}
	if e := s.Call(context.Background(), "echo", map[string]string{"x": strings.Repeat("x", MaxLine)}, nil); e == nil {
		t.Fatal("accepted oversized request")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if e := s.Call(ctx, "echo", nil, nil); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	if e := s.Call(context.Background(), "echo", nil, nil); e != nil {
		t.Fatal(e)
	}
}

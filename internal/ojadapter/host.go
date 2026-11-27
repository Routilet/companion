// Package ojadapter is a minimal child-process host. It has no network,
// installation, file-binding or GUI responsibilities.
package ojadapter

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
	"unicode/utf8"
)

const Protocol = "routilet-oj-adapter/1"
const MaxLine = 8 << 20
const handshakeTimeout = 5 * time.Second
const callTimeout = 60 * time.Second

var ErrClosed = errors.New("adapter session closed")
var ErrProtocol = errors.New("invalid adapter protocol response")

type Capabilities struct {
	Protocol string   `json:"protocol"`
	Adapter  string   `json:"adapter"`
	Version  string   `json:"version"`
	Methods  []string `json:"methods"`
}
type RemoteError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *RemoteError) Error() string { return "adapter: " + e.Code } // no untrusted payload in logs

type request struct {
	ID     uint64 `json:"id"`
	Method string `json:"method"`
	Params any    `json:"params"`
}
type response struct {
	ID     uint64          `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *RemoteError    `json:"error,omitempty"`
}

// Session is single-flight. Cancellation or framing failure invalidates the
// process, rather than allowing a late response to contaminate a new request.
// It never retries a request. Start a new session explicitly after failure.
type Session struct {
	cmd     *exec.Cmd
	in, out *os.File
	reader  *bufio.Reader
	gate    chan struct{}
	done    chan struct{}
	stop    sync.Once
	next    uint64 // accessed under gate
	caps    Capabilities
}

// Start accepts a trusted, already resolved absolute executable path. The
// caller must obtain it from verified installation metadata in a later phase.
// No searching PATH, downloading, invoking a shell, or auto-starting at boot.
func Start(ctx context.Context, executable, adapter string) (*Session, error) {
	if !filepath.IsAbs(executable) || adapter == "" {
		return nil, errors.New("adapter requires a trusted absolute executable path and ID")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	cmd := exec.Command(executable, "--stdio")
	cmd.Dir = filepath.Dir(executable)
	configure(cmd)
	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		stdinR.Close()
		stdinW.Close()
		return nil, err
	}
	cmd.Stdin = stdinR
	cmd.Stdout = stdoutW
	// Discard stderr, which might contain credentials or remote response bodies.
	// A failed process is reported as a generic transport error.
	cmd.Stderr = nil
	if err = cmd.Start(); err != nil {
		stdinR.Close()
		stdinW.Close()
		stdoutR.Close()
		stdoutW.Close()
		return nil, err
	}
	stdinR.Close()
	stdoutW.Close()
	s := &Session{cmd: cmd, in: stdinW, out: stdoutR, reader: bufio.NewReaderSize(stdoutR, 64<<10), gate: make(chan struct{}, 1), done: make(chan struct{})}
	go func() { _ = cmd.Wait(); close(s.done) }()
	hctx, cancel := context.WithTimeout(ctx, handshakeTimeout)
	defer cancel()
	var caps Capabilities
	if err = s.Call(hctx, "capabilities", map[string]string{"protocol": Protocol}, &caps); err != nil {
		s.Close()
		return nil, err
	}
	if caps.Protocol != Protocol || caps.Adapter != adapter || caps.Version == "" || len(caps.Version) > 128 || !validMethods(caps.Methods) {
		s.Close()
		return nil, ErrProtocol
	}
	s.caps = caps
	return s, nil
}
func validMethods(methods []string) bool {
	seen := map[string]bool{}
	if len(methods) > 128 {
		return false
	}
	for _, m := range methods {
		if m == "" || len(m) > 128 || seen[m] {
			return false
		}
		seen[m] = true
	}
	return seen["capabilities"]
}
func (s *Session) Capabilities() Capabilities {
	c := s.caps
	c.Methods = append([]string(nil), c.Methods...)
	return c
}

// Close is idempotent and may run concurrently with Call. Closing the pipes
// also releases blocked reads/writes. Wait reaps the child before returning.
func (s *Session) Close() {
	s.stop.Do(func() { _ = s.cmd.Process.Kill(); _ = s.in.Close(); _ = s.out.Close(); <-s.done })
}

func (s *Session) Call(ctx context.Context, method string, params any, out any) error {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.done:
		return ErrClosed
	case s.gate <- struct{}{}:
	}
	defer func() { <-s.gate }()
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case <-s.done:
		return ErrClosed
	default:
	}
	if method == "" {
		return errors.New("empty adapter method")
	}
	if params == nil {
		params = struct{}{}
	}
	raw, err := json.Marshal(params)
	if err != nil {
		return err
	}
	if len(raw) == 0 || raw[0] != '{' {
		return errors.New("adapter params must be a JSON object")
	}
	if len(raw) > MaxLine {
		return errors.New("adapter request too large")
	}
	s.next++
	id := s.next
	b, err := json.Marshal(request{id, method, json.RawMessage(raw)})
	if err != nil {
		return err
	}
	if len(b) > MaxLine {
		return errors.New("adapter request too large")
	}
	b = append(b, '\n')
	// The goroutine is joined on every path, including timeout or Close.
	type reply struct {
		r   response
		err error
	}
	result := make(chan reply, 1)
	go func() {
		if n, e := s.in.Write(b); e != nil || n != len(b) {
			result <- reply{err: ErrClosed}
			return
		}
		line, e := readLine(s.reader)
		if e != nil {
			result <- reply{err: e}
			return
		}
		var r response
		if decode(line, &r) != nil || r.ID != id || (len(r.Result) == 0) == (r.Error == nil) || (r.Error != nil && !validErrorCode(r.Error.Code)) {
			result <- reply{err: ErrProtocol}
			return
		}
		result <- reply{r: r}
	}()
	var a reply
	select {
	case <-ctx.Done():
		s.Close()
		<-result
		return ctx.Err()
	case a = <-result:
	}
	if ctx.Err() != nil {
		s.Close()
		return ctx.Err()
	}
	if a.err != nil {
		s.Close()
		return a.err
	}
	if a.r.Error != nil {
		return a.r.Error
	}
	if out != nil && decode(a.r.Result, out) != nil {
		s.Close()
		return ErrProtocol
	}
	return nil
}
func readLine(r *bufio.Reader) ([]byte, error) {
	var line []byte
	for {
		p, err := r.ReadSlice('\n')
		if len(line)+len(p) > MaxLine+1 {
			return nil, ErrProtocol
		}
		line = append(line, p...)
		if err == nil {
			return line[:len(line)-1], nil
		}
		if err != bufio.ErrBufferFull {
			if len(line) != 0 {
				return nil, ErrProtocol
			}
			return nil, ErrClosed
		}
	}
}
func decode(b []byte, v any) error {
	if !utf8.Valid(b) {
		return ErrProtocol
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	if d.Decode(new(any)) != io.EOF {
		return ErrProtocol
	}
	return nil
}

func validErrorCode(code string) bool {
	if len(code) == 0 || len(code) > 128 {
		return false
	}
	for _, r := range code {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_') {
			return false
		}
	}
	return true
}

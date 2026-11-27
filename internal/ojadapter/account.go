package ojadapter

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

// AccountPreviewEnv is a development-only trusted executable override;
// a provided Routilet manager always takes precedence.
const AccountPreviewEnv = "COMPANION_ACCODING_ADAPTER_PREVIEW"

type AccountState struct {
	State    string `json:"state"`
	ID       string `json:"id,omitempty"`
	Nickname string `json:"nickname,omitempty"`
}
type AccountView struct {
	AccountState
	Available   bool   `json:"available"`
	UnavailableReason string `json:"unavailableReason,omitempty"`
	CanRemember bool   `json:"canRemember"`
	Saved       bool   `json:"saved"`
	Warning     string `json:"warning,omitempty"`
}
type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type credentialStore interface {
	Load() (credentials, error)
	Save(credentials) error
	Delete() error
	Supported() bool
}

var errNoCredential = errors.New("no saved credential")
var errCredentialStore = errors.New("system credential store unavailable")

// Accounts owns one adapter session independently of local compiling/testing.
// Opening settings reads status only, never starts an adapter or contacts OJ.
type Accounts struct {
	mu         sync.Mutex
	executable string
	manager string
	managedAvailable bool
	unavailableReason string
	store      credentialStore
	session    *Session
	state      AccountState
	verified   bool
	closed     bool
	life       context.Context
	stop       context.CancelFunc
	saved      bool
}

func NewAccounts(path string) *Accounts {
	ctx, cancel := context.WithCancel(context.Background())
	return &Accounts{executable: path, store: systemCredentialStore(), life: ctx, stop: cancel}
}
// NewManagedAccounts uses only the manager injected by Routilet. A preview
// binary is accepted only when no manager was supplied, never as a fallback.
func NewManagedAccounts(manager, preview string) *Accounts {
 if manager=="" {return NewAccounts(preview)}
 a:=NewAccounts("");a.manager=manager
 a.unavailableReason="尚未检查 Routilet 中的 Adapter"
 return a
}

func (a *Accounts) available() bool {
 if a.manager!="" {return a.managedAvailable||a.session!=nil}
 if !filepath.IsAbs(a.executable) {
		return false
	}
	f, e := os.Stat(a.executable)
	return e == nil && f.Mode().IsRegular()
}
// checkAvailable runs an offline manager query only on explicit refresh or
// login/restore. Status and periodic tree cache reads never spawn processes.
func (a *Accounts) checkAvailable(ctx context.Context) bool {
 if a.manager=="" {return a.available()}
 err:=checkManaged(ctx,a.manager)
 a.managedAvailable=err==nil
 a.unavailableReason=""
 if err!=nil {a.unavailableReason=err.Error()}
 return a.managedAvailable
}
func (a *Accounts) CheckAvailability(ctx context.Context) AccountView {
 a.mu.Lock();defer a.mu.Unlock()
 if !a.closed {ctx,done:=a.context(ctx);defer done();a.checkAvailable(ctx)}
 return a.view()
}
func (a *Accounts) reset() {
	if a.session != nil {
		a.session.Close()
		a.session = nil
	}
	a.state = AccountState{State: "signed_out"}
	a.verified = false
	a.saved = false
}
func (a *Accounts) view() AccountView {
	if a.session != nil {
		select {
		case <-a.session.done:
			a.reset()
		default:
		}
	}
	state := a.state
	if state.State == "" {
		state.State = "signed_out"
	}
	if state.State == "signed_in" && !a.verified {
		state.State = "unverified"
	}
	v:=AccountView{AccountState: state, Available: !a.closed && a.available(), CanRemember: a.store.Supported(), Saved: a.saved}
 if !v.Available {v.UnavailableReason=a.unavailableReason;if v.UnavailableReason==""{v.UnavailableReason="请通过 Routilet 安装并启用 Accoding Adapter；本地功能不受影响。"}}
 return v
}

// Status deliberately does not open the system keychain (which can prompt) or start a process.
func (a *Accounts) Status() AccountView { a.mu.Lock(); defer a.mu.Unlock(); return a.view() }
func (a *Accounts) ensure(ctx context.Context) error {
	if a.closed || !a.available() {
		return errors.New("Accoding Adapter 不可用")
	}
	if a.session != nil {
		select {
		case <-a.session.done:
			a.reset()
		default:
			return nil
		}
	}
	var s *Session
 var e error
 if a.manager!="" {s,e=startCommand(ctx,a.manager,"accoding",true,"--adapter-stdio","accoding-adapter")}else{s,e=Start(ctx,a.executable,"accoding")}
	if e != nil {
  if a.manager!=""{a.managedAvailable=false;a.unavailableReason="无法启动已安装的 Adapter，请在 Routilet 检查安装或更新"}
  return errors.New("无法启动 Accoding Adapter")
 }
	caps := s.Capabilities()
	m := map[string]bool{}
	for _, k := range caps.Methods {
		m[k] = true
	}
	if !m["account"] || !m["login"] || !m["logout"] {
		s.Close()
		return errors.New("Adapter 不支持账号管理")
	}
	a.session = s
	return nil
}
func (a *Accounts) call(ctx context.Context, method string, p any) (AccountState, error) {
	var state AccountState
	if err := a.session.Call(ctx, method, p, &state); err != nil {
		var remote *RemoteError
		if !errors.As(err, &remote) {
			a.reset()
			return state, errors.New("Adapter 通信中断，请重新登录")
		}
		switch remote.Code {
		case "auth", "forbidden":
			a.state = AccountState{State: "signed_out"}
			a.verified = false
			return state, errors.New("登录失败、会话已过期或无权限")
		case "network", "server", "rate", "cancelled":
			a.verified = false
			return state, errors.New("无法确认账号状态，请稍后重试")
		default:
			a.verified = false
			return state, errors.New("账号操作失败")
		}
	}
	valid := state.State == "signed_out" && state.ID == "" && state.Nickname == "" || state.State == "signed_in" && len(state.ID) > 0 && len(state.ID) <= 64 && state.ID != "0" && len(state.Nickname) <= 4096
	if valid && state.State == "signed_in" {
		for _, r := range state.ID {
			if r < '0' || r > '9' {
				valid = false
			}
		}
	}
	if method == "account" && a.state.ID != "" && state.State == "signed_in" && state.ID != a.state.ID {
		valid = false
	}
	if !valid || (method == "login" && state.State != "signed_in") || (method == "logout" && state.State != "signed_out") {
		a.reset()
		return AccountState{}, errors.New("Adapter 返回无效账号状态")
	}
	a.state = state
	a.verified = state.State == "signed_in"
	return state, nil
}
func (a *Accounts) login(ctx context.Context, c credentials, remember bool) (AccountView, error) {
	if c.Username == "" || c.Password == "" || len(c.Username) > 1024 || len(c.Password) > 4096 {
		return a.view(), errors.New("请输入有效账号和密码")
	}
	if a.closed || !a.checkAvailable(ctx) {
  return a.view(), errors.New("Accoding Adapter 不可用，请在 Routilet 检查安装与启用状态")
 }
	if remember && !a.store.Supported() {
		return a.view(), errCredentialStore
	}
	// Clear previous identity and its saved credential before switching. If the
	// keychain cannot be cleared, do not start a different account session.
	if a.store.Supported() {
		if e := a.store.Delete(); e != nil && !errors.Is(e, errNoCredential) {
			return a.view(), errCredentialStore
		}
	}
	a.reset()
	if err := a.ensure(ctx); err != nil {
		return a.view(), err
	}
	if _, err := a.call(ctx, "login", c); err != nil {
		return a.view(), err
	}
	v := a.view()
	if remember {
		if e := a.store.Save(c); e != nil {
			v.Warning = "已登录，但系统凭据保存失败"
		} else {
			v.Saved = true
			a.saved = true
		}
	}
	return v, nil
}
func (a *Accounts) Login(ctx context.Context, user, password string, remember bool) (AccountView, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	ctx, done := a.context(ctx)
	defer done()
	return a.login(ctx, credentials{user, password}, remember)
}

// Restore is explicit, never called at startup. On a transient failure keep the
// stored credential so the user can retry or explicitly clear it.
func (a *Accounts) Restore(ctx context.Context) (AccountView, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	ctx, done := a.context(ctx)
	defer done()
	if a.closed || !a.checkAvailable(ctx) {
  return a.view(), errors.New("Accoding Adapter 不可用，请在 Routilet 检查安装与启用状态")
 }
	if !a.store.Supported() {
		return a.view(), errCredentialStore
	}
	c, e := a.store.Load()
	if e != nil {
		return a.view(), errors.New("未找到可用的已保存凭据")
	}
	a.reset()
	if e = a.ensure(ctx); e != nil {
		return a.view(), e
	}
	_, e = a.call(ctx, "login", c)
	v := a.view()
	if e != nil {
		return v, e
	}
	v.Saved = true
	a.saved = true
	return v, nil
}
func (a *Accounts) Refresh(ctx context.Context) (AccountView, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	ctx, done := a.context(ctx)
	defer done()
	if a.session == nil {
		return a.view(), nil
	}
	_, e := a.call(ctx, "account", map[string]bool{"refresh": true})
	return a.view(), e
}

// Logout clears local session even if the remote transport is gone, and
// attempts to delete persisted credentials. Failure is surfaced, not hidden.
func (a *Accounts) Logout() (AccountView, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.reset()
	v := a.view()
	if a.store.Supported() {
		if e := a.store.Delete(); e != nil && !errors.Is(e, errNoCredential) {
			return v, errors.New("已注销，但系统凭据删除失败；请重试注销")
		}
	}
	return v, nil
}
func (a *Accounts) context(parent context.Context) (context.Context, func()) {
	ctx, cancel := context.WithCancel(parent)
	stop := context.AfterFunc(a.life, cancel)
	if a.life.Err() != nil {
		cancel()
	}
	return ctx, func() { stop(); cancel() }
}
func (a *Accounts) Close() { a.stop(); a.mu.Lock(); defer a.mu.Unlock(); a.closed = true; a.reset() }

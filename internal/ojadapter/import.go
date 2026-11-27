package ojadapter

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"
)

type Target struct {
	Kind      string `json:"kind"`
	ContestID string `json:"contestId,omitempty"`
	ProblemID string `json:"problemId"`
	Service   string `json:"service,omitempty"`
}
type Sample struct {
	Input    string `json:"input"`
	Expected string `json:"expected"`
}
type Problem struct {
	Target    Target   `json:"target"`
	Title     string   `json:"title"`
	Label     string   `json:"label"`
	SourceURL string   `json:"sourceUrl"`
	Samples   []Sample `json:"samples"`
	Warnings  []string `json:"warnings"`
	Special   bool     `json:"special"`
}
type ImportResult struct {
	AccountID string    `json:"accountId"`
	Title     string    `json:"title"`
	Problems  []Problem `json:"problems"`
}

var numberID = regexp.MustCompile(`^[1-9][0-9]{0,63}$`)
var letterLabel = regexp.MustCompile(`^[A-Z]{1,4}$`)

func (t Target) URL() string {
	if !numberID.MatchString(t.ProblemID) {
		return ""
	}
	if t.Kind == "contest" && numberID.MatchString(t.ContestID) && t.Service == "" {
		return "https://accoding.buaa.edu.cn/contest-ng/index.html#/" + t.ContestID
	}
	if t.Kind != "problemset" || t.ContestID != "" {
		return ""
	}
	host := "https://accoding.buaa.edu.cn"
	if t.Service == "admin" {
		host += ":4000"
	} else if t.Service != "" {
		return ""
	}
	return host + "/problem/" + t.ProblemID + "/index"
}
func (p Problem) Filename(ext string) (string, error) {
	if ext != ".c" && ext != ".cpp" {
		return "", errors.New("无效编程语言")
	}
	if p.Target.Kind == "contest" && letterLabel.MatchString(p.Label) {
		return p.Label + ext, nil
	}
	if p.Target.Kind == "problemset" && numberID.MatchString(p.Target.ProblemID) {
		return "problem-" + p.Target.ProblemID + ext, nil
	}
	return "", errors.New("无效题序")
}
func validText(s string, n int) bool {
	return len(s) <= n && utf8.ValidString(s) && !strings.ContainsRune(s, 0)
}
func (r ImportResult) Validate(account string) error {
	if !numberID.MatchString(account) || r.AccountID != account || r.Title == "" || !validText(r.Title, 4096) || len(r.Problems) == 0 || len(r.Problems) > 200 {
		return errors.New("导入结果或账号无效")
	}
	seen := map[Target]bool{}
	names := map[string]bool{}
	first := r.Problems[0].Target
	for _, p := range r.Problems {
		n, e := p.Filename(".c")
		if e != nil || names[n] || seen[p.Target] || p.Target.URL() == "" || p.SourceURL != p.Target.URL() || p.Target.Kind != first.Kind || p.Target.ContestID != first.ContestID || p.Title == "" || !validText(p.Title, 4096) || !validText(p.Label, 128) || len(p.Warnings) > 64 || len(p.Samples) > 32 || p.Special && len(p.Samples) != 0 || p.Target.Kind == "problemset" && len(r.Problems) != 1 {
			return errors.New("导入题目数据无效或重复")
		}
		seen[p.Target] = true
		names[n] = true
		for _, w := range p.Warnings {
			if !validText(w, 4096) {
				return errors.New("导入警告无效")
			}
		}
		for _, s := range p.Samples {
			if !validText(s.Input, 1<<20) || !validText(s.Expected, 1<<20) {
				return errors.New("公开样例过大或无效")
			}
		}
	}
	return nil
}

// Import is read-only. It cannot create files or start a new authenticated session.
func (a *Accounts) Import(ctx context.Context, link string) (ImportResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	ctx, done := a.context(ctx)
	defer done()
	if a.closed || a.session == nil || a.state.ID == "" {
		return ImportResult{}, errors.New("请先在设置中登录 Accoding")
	}
	id := a.state.ID
	var r ImportResult
	if err := a.session.Call(ctx, "import", map[string]string{"url": link}, &r); err != nil {
		var remote *RemoteError
		if !errors.As(err, &remote) {
			a.reset()
			return r, errors.New("Adapter 通信中断，请重新登录")
		}
		if remote.Code == "auth" {
			a.reset()
		}
		switch remote.Code {
		case "auth":
			return r, errors.New("登录已失效，请重新登录")
		case "forbidden":
			return r, errors.New("当前账号无权读取题目")
		case "invalid_params":
			return r, errors.New("无法识别 Accoding 链接")
		case "network", "rate", "server", "cancelled":
			return r, errors.New("无法获取远程数据，请稍后重试")
		default:
			return r, errors.New("Adapter 导入失败或接口不兼容")
		}
	}
	if e := r.Validate(id); e != nil {
		a.reset()
		return ImportResult{}, e
	}
	a.verified = true
	return r, nil
}

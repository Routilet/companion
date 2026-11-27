package ojadapter

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Binding struct {
	SchemaVersion int    `json:"schemaVersion"`
	Adapter       string `json:"adapter"`
	Title         string `json:"title"`
	Label         string `json:"label"`
	SourceURL     string `json:"sourceUrl"`
	Target        Target `json:"target"`
 ContestTitle string `json:"contestTitle,omitempty"`
}

func (p Problem) Binding() Binding {
	return Binding{SchemaVersion:1,Adapter:"accoding",Title:p.Title,Label:p.Label,SourceURL:p.SourceURL,Target:p.Target}
}
func (b Binding) validate() error {
	if b.SchemaVersion != 1 || b.Adapter != "accoding" || b.Target.URL() == "" || b.SourceURL != b.Target.URL() || b.Title == "" || !validText(b.Title, 4096) || !validText(b.Label, 128) || !validText(b.ContestTitle,4096) {
		return errors.New("题目关联文件无效")
	}
	return nil
}
func bindingPath(source string, create bool) (string, error) {
	if !filepath.IsAbs(source) || filepath.Base(source) == "." {
		return "", errors.New("无效源码路径")
	}
	switch strings.ToLower(filepath.Ext(source)) {
	case ".c", ".cpp", ".cc", ".cxx":
	default:
		return "", errors.New("无效源码类型")
	}
	p := filepath.Dir(source)
	for _, n := range []string{"", ".companion", "bindings"} {
		if n != "" {
			p = filepath.Join(p, n)
		}
		st, e := os.Lstat(p)
		if n != "" && create && errors.Is(e, os.ErrNotExist) {
			if e = os.Mkdir(p, 0700); e != nil && !errors.Is(e, os.ErrExist) {
				return "", e
			}
			st, e = os.Lstat(p)
		}
		if e != nil {
			return "", e
		}
		if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("关联目录不是普通目录")
		}
	}
	return filepath.Join(p, filepath.Base(source)+".json"), nil
}
func LoadBinding(source string) (*Binding, error) {
	p, e := bindingPath(source, false)
	if errors.Is(e, os.ErrNotExist) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	st, e := os.Lstat(p)
	if errors.Is(e, os.ErrNotExist) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	if !st.Mode().IsRegular() {
		return nil, errors.New("关联文件不是普通文件")
	}
	f, e := os.Open(p)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	raw, e := io.ReadAll(io.LimitReader(f, 65537))
	if e != nil || len(raw) > 65536 {
		return nil, errors.New("关联文件过大或读取失败")
	}
	var b Binding
	if json.Unmarshal(raw, &b) != nil || b.validate() != nil {
		return nil, errors.New("关联文件损坏；未修改")
	}
	return &b, nil
}

// No overwrite, even if an external writer creates the destination concurrently.
func SaveBinding(source string, b Binding) error {
	if e := b.validate(); e != nil {
		return e
	}
	p, e := bindingPath(source, true)
	if e != nil {
		return e
	}
	raw, e := json.MarshalIndent(b, "", "  ")
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(p), ".binding-*")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	_, e = f.Write(append(raw, '\n'))
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	return os.Link(f.Name(), p)
}
func RemoveBinding(source string) error {
	b, e := LoadBinding(source)
	if e != nil || b == nil {
		return e
	}
	p, e := bindingPath(source, false)
	if e != nil {
		return e
	}
	return os.Remove(p)
}

// RenameBound reserves the destination metadata without replacing it; the caller
// performs existing source/case rename. Failures preserve original metadata.
func RenameBound(old, next string, rename func() error) error {
	if old == next {
		return rename()
	}
	if b, e := LoadBinding(next); e != nil {
		return e
	} else if b != nil {
		return errors.New("目标存在旧题目关联")
	}
	b, e := LoadBinding(old)
	if e != nil {
		return e
	}
	if b == nil {
		return rename()
	}
	if e = SaveBinding(next, *b); e != nil {
		return e
	}
	if e = rename(); e != nil {
		if re := RemoveBinding(next); re != nil {
			return errors.Join(e, re)
		}
		return e
	}
	if e = RemoveBinding(old); e != nil {
		return errors.New("源码已重命名，旧关联仍保留，请检查")
	}
	return nil
}

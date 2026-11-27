package ojadapter

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// A plan is a local preview. Names are derived from validated platform IDs,
// never from arbitrary titles. Existing files can only be attached explicitly.
type PlanRow struct {
	Name     string   `json:"name"`
	Title    string   `json:"title"`
	Action   string   `json:"action"`
	Samples  int      `json:"samples"`
	Warnings []string `json:"warnings"`
 Document string `json:"document,omitempty"`
	problem  Problem
	info     os.FileInfo
	revision string
}
type Plan struct {
	Title     string    `json:"title"`
	Directory string    `json:"directory"`
	Rows      []PlanRow `json:"rows"`
	AccountID string    `json:"-"`
	Attach    string    `json:"-"`
 Archive bool `json:"-"`
 FolderName string `json:"folderName,omitempty"`
 CanRename bool `json:"canRename,omitempty"`
}

// CaseRevision is supplied by the existing Companion case store for an existing source.
type CaseRevision func(source string) (string, error)

func checkNew(source string) error {
	if _, e := os.Lstat(source); e == nil {
		return errors.New("源码已存在")
	} else if !errors.Is(e, os.ErrNotExist) {
		return e
	}
	if b, e := LoadBinding(source); e != nil {
		return e
	} else if b != nil {
		return errors.New("存在旧题目关联")
	}
	p := filepath.Dir(source)
	for _, n := range []string{".companion", "cases"} {
		p = filepath.Join(p, n)
		st, e := os.Lstat(p)
		if errors.Is(e, os.ErrNotExist) {
			return nil
		}
		if e != nil {
			return e
		}
		if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
			return errors.New("测试数据目录无效")
		}
	}
	if _, e := os.Lstat(filepath.Join(p, filepath.Base(source)+".json")); e == nil {
		return errors.New("存在旧测试数据")
	} else if !errors.Is(e, os.ErrNotExist) {
		return e
	}
	return nil
}
func BuildPlan(result ImportResult, dir, ext, attach string, revision CaseRevision) (Plan, error) {
	if e := result.Validate(result.AccountID); e != nil {
		return Plan{}, e
	}
	if !filepath.IsAbs(dir) {
		return Plan{}, errors.New("无效目录")
	}
	if attach != "" && (len(result.Problems) != 1 || filepath.Dir(attach) != dir || !filepath.IsAbs(attach)) {
		return Plan{}, errors.New("只允许将单题关联到当前文件")
	}
	if ext != ".c" && ext != ".cpp" {
		return Plan{}, errors.New("无效语言")
	}
	out := Plan{Title: result.Title, Directory: dir, AccountID: result.AccountID, Attach: attach, Rows: []PlanRow{}}
	for _, p := range result.Problems {
		n, e := p.Filename(ext)
		if e != nil {
			return Plan{}, e
		}
		row := PlanRow{Name: n, Title: p.Title, Action: "create", Samples: len(p.Samples), Warnings: append([]string{}, p.Warnings...), problem: p}
		if attach == "" {
			e = checkNew(filepath.Join(dir, n))
		} else {
			row.Name = filepath.Base(attach)
			row.Action = "attach"
			row.info, e = os.Lstat(attach)
			if e == nil && (!row.info.Mode().IsRegular() || !strings.EqualFold(filepath.Ext(attach), ext) && !(ext == ".cpp" && (strings.EqualFold(filepath.Ext(attach), ".cc") || strings.EqualFold(filepath.Ext(attach), ".cxx")))) {
				e = errors.New("当前文件类型不匹配")
			}
			if e == nil {
				var b *Binding
				b, e = LoadBinding(attach)
				if e == nil && b != nil {
					e = errors.New("当前文件已有题目关联")
				}
			}
			if e == nil {
				row.revision, e = revision(attach)
				if e == nil && row.revision != "missing" {
					row.Samples = 0
					row.Warnings = append(row.Warnings, "保留全部现有测试数据，不自动导入公开样例")
				}
			}
		}
		if e != nil {
			row.Action = "skip"
			row.Samples = 0
			row.Warnings = append(row.Warnings, e.Error())
		}
		out.Rows = append(out.Rows, row)
	}
	return out, nil
}

// Recheck immediately before any write; stale plans fail closed.
func (p Plan) Recheck(row PlanRow, revision CaseRevision) error {
	path := filepath.Join(p.Directory, row.Name)
	if p.Archive&&row.Document!=""{if e:=archivePath(p.Directory,path);e!=nil{return e};if _,e:=os.Lstat(filepath.Join(filepath.Dir(path),row.Document));e==nil||!os.IsNotExist(e){return errors.New("题面路径已存在或无效")}}
 switch row.Action {
	case "create":
        if p.Archive {if e:=archivePath(p.Directory,path);e!=nil{return e};if row.Document!=""{if _,e:=os.Lstat(filepath.Join(filepath.Dir(path),row.Document));e==nil||!os.IsNotExist(e){return errors.New("题面路径已存在或无效")}}}
		return checkNew(path)
    case "repair":
        if e:=archivePath(p.Directory,path);e!=nil{return e}
        st,e:=os.Lstat(path);if e!=nil||row.info==nil||!st.Mode().IsRegular()||!os.SameFile(st,row.info){return errors.New("源码已变化")}
        b,e:=LoadBinding(path);if e!=nil||b==nil||b.Target!=row.problem.Target{return errors.New("题目关联已变化")}
        r,e:=revision(path);if e!=nil||r!=row.revision{return errors.New("测试数据已变化")};return nil
	case "attach":
		st, e := os.Lstat(path)
		if e != nil || row.info == nil || !st.Mode().IsRegular() || !os.SameFile(row.info, st) {
			return errors.New("当前源码已被替换")
		}
		if b, e := LoadBinding(path); e != nil || b != nil {
			return errors.New("题目关联已变化")
		}
		r, e := revision(path)
		if e != nil || r != row.revision {
			return errors.New("测试数据已变化")
		}
		return nil
	default:
		return errors.New("该题不可导入")
	}
}
func (r PlanRow) Problem() Problem { return r.problem }
func (r PlanRow) Revision() string {
	if r.Action == "create" {
		return "missing"
	}
	return r.revision
}

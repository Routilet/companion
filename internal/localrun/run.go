// Package localrun runs explicitly selected, ordinary single-file C programs.
// It has no dependency on the GUI, OJ, or user metadata.
package localrun

import (
 "bytes"
 "context"
 "crypto/sha256"
 "crypto/rand"
 "encoding/json"
 "encoding/hex"
 "errors"
 "fmt"
 "io"
 "os"
 "os/exec"
 "path/filepath"
 "regexp"
 "runtime"
 "strings"
 "sync"
 "time"
)

const maxSource = 2 * 1024 * 1024
const maxInput = 1024 * 1024
const maxOutput = 4 * 1024 * 1024

type Source struct {
 Path string `json:"path"`
 Directory string `json:"directory"`
 Name string `json:"name"`
}

type Process struct {
 Stdout string `json:"stdout"`
 Stderr string `json:"stderr"`
 ExitCode int `json:"exitCode"`
 ElapsedMS int64 `json:"elapsedMs"`
 State string `json:"state"`
}

type Result struct {
 Source Source `json:"source"`
 SourceDigest string `json:"sourceDigest"`
 Compiler string `json:"compiler"`
 State string `json:"state"`
 Compilation Process `json:"compilation"`
 Execution *Process `json:"execution,omitempty"`
 Stale bool `json:"stale"`
}

func Open(path string) (Source,error) {
 p,e:=filepath.Abs(path)
 if e!=nil{return Source{},e}
 if !sourceExtension(p){return Source{},errors.New("请选择普通 C/C++ 源文件")}
 st,e:=os.Lstat(p)
 if e!=nil{return Source{},e}
 if !st.Mode().IsRegular(){return Source{},errors.New("请选择普通 C 源文件，而不是目录或链接")}
 return Source{Path:p,Directory:filepath.Dir(p),Name:filepath.Base(p)},nil
}

func readSource(s Source)([]byte,error){
 f,e:=os.Open(s.Path);if e!=nil{return nil,e};defer f.Close()
 st,e:=f.Stat();if e!=nil{return nil,e};if !st.Mode().IsRegular(){return nil,errors.New("源文件不是普通文件")}
 b,e:=io.ReadAll(io.LimitReader(f,maxSource+1));if e!=nil{return nil,e}
 if len(b)>maxSource{return nil,errors.New("源文件超过 2 MiB")}
 return b,nil
}

// Conservative gate for this slice, not a C preprocessor or sandbox.
// Reject non-literal and local include directives rather than misrepresent
// them as independently compilable source. Ordinary macros are allowed.
var include = regexp.MustCompile("(?m)^[ \\t]*#[ \\t]*include\\b([^\\r\\n]*)")
var systemInclude = regexp.MustCompile("^[ \\t]*<[A-Za-z0-9_+./-]+>[ \\t]*(?://.*)?$")
func checkSource(b []byte) error {
 s:=strings.ReplaceAll(strings.ReplaceAll(string(b),"\\\r\n",""),"\\\n","")
 for _,m:=range include.FindAllStringSubmatch(s,-1){
  if !systemInclude.MatchString(m[1])||strings.Contains(m[1],"..")||strings.Contains(m[1],"</"){
   return errors.New("当前暂不支持本地、宏生成或复杂 include（C++ 不支持 Exam Kit）")
  }
 }
 return nil
}
func compilerPath(explicit string)(string,error){return compilerFor(explicit,false)}
func compilerFor(explicit string,cpp bool)(string,error){
 names:=[]string{explicit}
 if explicit==""{
  if cpp {names=[]string{os.Getenv("CXX")};if runtime.GOOS=="windows"{names=append(names,"g++")}else{names=append(names,"c++")}}else{names=[]string{os.Getenv("CC")};if runtime.GOOS=="windows"{names=append(names,"gcc")}else{names=append(names,"cc")}}
 }
 for _,n:=range names{if n==""{continue};p,e:=exec.LookPath(n);if e==nil{return filepath.Abs(p)}}
 return "",errors.New("未找到 C/C++ 编译器，请安装 GCC/Clang 或在编译器栏填写现有可执行文件路径")
}
func digest(b []byte)string{h:=sha256.Sum256(b);return hex.EncodeToString(h[:])}

type capture struct{ mu sync.Mutex; out,err bytes.Buffer; limit int; hit bool; stop context.CancelFunc }
type sink struct{ c *capture; stderr bool }
func(s sink)Write(p []byte)(int,error){
 c:=s.c;c.mu.Lock();defer c.mu.Unlock()
 left:=c.limit-c.out.Len()-c.err.Len()
 n:=len(p);if n>left{n=left;c.hit=true;c.stop()}
 if s.stderr{_,_=c.err.Write(p[:n])}else{_,_=c.out.Write(p[:n])}
 return len(p),nil
}

func process(parent context.Context,exe string,args []string,cwd,input string,env []string,timeout time.Duration) Process{
 ctx,stop:=context.WithTimeout(parent,timeout);defer stop()
 start:=time.Now()
 c:=&capture{limit:maxOutput,stop:stop}
 cmd:=exec.Command(exe,args...)
 cmd.Dir=cwd;cmd.Env=env;cmd.Stdin=strings.NewReader(input)
 cmd.Stdout=sink{c:c};cmd.Stderr=sink{c:c,stderr:true}
 configureProcess(cmd)
 cmd.WaitDelay=500*time.Millisecond
 if ctx.Err()!=nil{return Process{ExitCode:-1,State:"cancelled"}}
 err:=cmd.Start()
 if err==nil{
  owner:=ownProcess(cmd)
  done:=make(chan error,1)
  go func(){done<-cmd.Wait()}()
  select{
  case err=<-done:
   owner.stop()
  case <-ctx.Done():
   owner.stop()
   err=<-done
  }
  owner.close()
 }
 c.mu.Lock();defer c.mu.Unlock()
 r:=Process{Stdout:c.out.String(),Stderr:c.err.String(),ExitCode:-1,ElapsedMS:time.Since(start).Milliseconds(),State:"ok"}
 if cmd.ProcessState!=nil{r.ExitCode=cmd.ProcessState.ExitCode()}
 switch{
 case c.hit:r.State="output_limit"
 case parent.Err()!=nil:r.State="cancelled"
 case errors.Is(ctx.Err(),context.DeadlineExceeded):r.State="timeout"
 case err!=nil:
  r.State="failed"
  if r.Stderr==""{r.Stderr=err.Error()}
 }
 return r
}

func childEnv(compiler string)[]string{
 if runtime.GOOS!="windows"{return nil}
 env:=os.Environ()
 for i,v:=range env{if strings.EqualFold(strings.SplitN(v,"=",2)[0],"PATH"){
  env[i]="Path="+filepath.Dir(compiler)+string(os.PathListSeparator)+strings.SplitN(v,"=",2)[1];return env
 }}
 return append(env,"Path="+filepath.Dir(compiler))
}
func artifactArg(cwd,p string)string{
 r,e:=filepath.Rel(cwd,p);if e!=nil{return p};return "."+string(filepath.Separator)+r
}

type CaseResult struct {
 Index int `json:"index"`
 Input string `json:"input"`
 Expected *string `json:"expected"`
 Process Process `json:"process"`
 State string `json:"state"`
 Difference *Difference `json:"difference,omitempty"`
}
type BatchResult struct {
 RunID string `json:"runId"`
 Source Source `json:"source"`
 SourceDigest string `json:"sourceDigest"`
 PreparedDigest string `json:"preparedDigest"`
 ResourceDigest string `json:"resourceDigest,omitempty"`
 KitVersion string `json:"kitVersion,omitempty"`
 Components []string `json:"components,omitempty"`
 ToolchainDigest string `json:"toolchainDigest"`
 CaseRevision string `json:"caseRevision"`
 Compiler string `json:"compiler"`
 State string `json:"state"`
 Compilation Process `json:"compilation"`
 Diagnostics []Diagnostic `json:"diagnostics"`
 Cases []CaseResult `json:"cases"`
 Stale bool `json:"stale"`
}
var cflags=[]string{"-std=c99","-O2","-Wall","-Wextra"}
func flagsFor(path string)[]string{if isCPP(path){return []string{"-std=c++17","-O2","-Wall","-Wextra"}};return cflags}
func toolchainDigest(cc,path string)string{b,_:=json.Marshal([]any{cc,flagsFor(path),"-lm"});return digest(b)}
func Current(r BatchResult,compiler string)bool{
 s,e:=Open(r.Source.Path);if e!=nil{return false}
 b,e:=readSource(s);if e!=nil{return false}
 p,e:=PrepareSource(s,b);if e!=nil||p.SourceDigest!=r.SourceDigest||p.PreparedDigest!=r.PreparedDigest||p.ResourceDigest!=r.ResourceDigest{return false}
 cc,e:=compilerFor(compiler,isCPP(s.Path));if e!=nil||toolchainDigest(cc,s.Path)!=r.ToolchainDigest{return false}
 cases,e:=LoadCases(s.Path);return e==nil&&cases.Revision==r.CaseRevision
}
type Progress struct {
 Phase string `json:"phase"`
 Completed int `json:"completed"`
 Total int `json:"total"`
 RunID string `json:"runId"`
}
func batch(ctx context.Context,path,compiler string,cs CaseSet,selection int)(BatchResult,error){
 return batchProgress(ctx,path,compiler,cs,selection,nil)
}
func batchProgress(ctx context.Context,path,compiler string,cs CaseSet,selection int,progress func(Progress))(result BatchResult,err error){
 if e:=validateCases(cs.Cases);e!=nil{return result,e}
 s,e:=Open(path);if e!=nil{return result,e}
 b,e:=readSource(s);if e!=nil{return result,e}
 prepared,e:=PrepareSource(s,b);if e!=nil{return result,e}
 cc,e:=compilerFor(compiler,isCPP(s.Path));if e!=nil{return result,e}
 var id [16]byte;if _,e=rand.Read(id[:]);e!=nil{return result,e}
 result=BatchResult{RunID:hex.EncodeToString(id[:]),Source:s,SourceDigest:prepared.SourceDigest,PreparedDigest:prepared.PreparedDigest,ResourceDigest:prepared.ResourceDigest,KitVersion:prepared.KitVersion,Components:prepared.Components,Compiler:cc,ToolchainDigest:toolchainDigest(cc,s.Path),CaseRevision:cs.Revision,State:"compilation_failed",Cases:[]CaseResult{},Diagnostics:[]Diagnostic{}}
 defer func(){result.Stale=!Current(result,compiler)}()
 if len(cs.Cases)==0{cs.Cases=[]Case{{Input:""}};selection=0}
 if selection< -1||selection>=len(cs.Cases){return result,errors.New("无效的测试编号")}
 total:=len(cs.Cases);if selection>=0{total=1}
 report:=func(phase string,completed int){if progress!=nil{progress(Progress{phase,completed,total,result.RunID})}}
 report("compiling",0)
 dir,e:=os.MkdirTemp("","companion-run-");if e!=nil{return result,e}
 defer func(){if e:=os.RemoveAll(dir);e!=nil&&err==nil{err=fmt.Errorf("清理临时目录失败: %w",e)}}()
 snapshot:=filepath.Join(dir,"source"+strings.ToLower(filepath.Ext(s.Path)));program:=filepath.Join(dir,"program")
 if runtime.GOOS=="windows"{program+=".exe"}
 if e=os.WriteFile(snapshot,prepared.Bytes,0600);e!=nil{return result,e}
 env:=childEnv(cc)
 args:=append(append([]string{},flagsFor(s.Path)...),artifactArg(s.Directory,snapshot),"-o",artifactArg(s.Directory,program),"-lm")
 result.Compilation=process(ctx,cc,args,s.Directory,"",env,30*time.Second)
 result.Diagnostics=diagnostics(result.Compilation.Stderr,snapshot,s.Path)
 mapPreparedDiagnostics(result.Diagnostics,prepared,s.Path)
 if result.Compilation.State!="ok"{
  if result.Compilation.State!="failed"{result.State=result.Compilation.State}
  return result,nil
 }
 result.State="completed"
 report("running",0)
 for i,c:=range cs.Cases{
  if selection>=0&&selection!=i{continue}
  if ctx.Err()!=nil{result.State="cancelled";break}
  p:=process(ctx,program,nil,s.Directory,c.Input,env,2*time.Second)
  r:=CaseResult{Index:i,Input:c.Input,Expected:c.Expected,Process:p,State:p.State}
  if p.State=="ok"{
   if c.Expected==nil{r.State="ran"}else if r.Difference=Compare(*c.Expected,p.Stdout);r.Difference==nil{r.State="passed"}else{r.State="mismatch"}
  }else if p.State=="failed"{r.State="runtime_failed"}
  result.Cases=append(result.Cases,r)
  report("running",len(result.Cases))
  if p.State=="cancelled"{result.State="cancelled";break}
 }
 return result,nil
}
// RunBatch uses an immutable copy of the persisted cases and rejects a stale UI revision.
func RunBatch(ctx context.Context,path,compiler,revision string,selection int)(BatchResult,error){
 return RunBatchProgress(ctx,path,compiler,revision,selection,nil)
}
func RunBatchProgress(ctx context.Context,path,compiler,revision string,selection int,progress func(Progress))(BatchResult,error){
 cs,e:=LoadCases(path);if e!=nil{return BatchResult{},e}
 if cs.Revision!=revision{return BatchResult{},errors.New("测试数据已变化，请重新读取")}
 return batchProgress(ctx,path,compiler,cs,selection,progress)
}
// Run is retained for the first-slice regression tests and single input callers.
func Run(ctx context.Context,path,input,compiler string)(Result,error){
 cs,e:=LoadCases(path);if e!=nil{return Result{},e}
 cs.Cases=[]Case{{Input:input}}
 b,e:=batch(ctx,path,compiler,cs,0)
 r:=Result{Source:b.Source,SourceDigest:b.SourceDigest,Compiler:b.Compiler,State:b.State,Compilation:b.Compilation,Stale:b.Stale}
 if len(b.Cases)>0{r.Execution=&b.Cases[0].Process;r.State=b.Cases[0].State}
 return r,e
}

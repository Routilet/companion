package localrun

import (
 "bytes"
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "io"
 "os"
 "os/exec"
 "path/filepath"
 "runtime"
 "strings"
 "time"
)

const ObserverPreviewEnv="COMPANION_OBSERVER_PACKAGE"
type observerTarget struct{
 Executable string `json:"executable"`
 SHA256 string `json:"sha256"`
 Status string `json:"status"`
}
type observerManifest struct{
 SchemaVersion int `json:"schemaVersion"`
 Version string `json:"version"`
 SessionSchemaVersions []int `json:"sessionSchemaVersions"`
 Platforms map[string]observerTarget `json:"platforms"`
}
type ObserverInfo struct{
 Version string `json:"version"`
 Executable string `json:"executable"`
 Backend string `json:"backend"`
 Error string `json:"error,omitempty"`
}
type observerCapabilities struct{
 Version string `json:"version"`
 Platform string `json:"platform"`
 Arch string `json:"arch"`
 Backends []string `json:"backends"`
 SessionSchemaVersions []int `json:"sessionSchemaVersions"`
 SessionFeatures []string `json:"sessionFeatures"`
}
func hasString(list []string,want string)bool{for _,v:=range list{if v==want{return true}};return false}
func hasV1(list []int)bool{for _,v:=range list{if v==1{return true}};return false}
func observerPlatform()(string,string,error){
 if runtime.GOOS=="windows"&&runtime.GOARCH=="amd64"{return "win32-x64","gdb",nil}
 if runtime.GOOS=="darwin"&&runtime.GOARCH=="arm64"{return "darwin-arm64","lldb",nil}
 return "","",errors.New("Observer 0.2 GUI 仅支持 Windows x64 和 Apple Silicon macOS")
}
// Resolve the official package's manifest, never a guessed bin path.
// A checksum verifies package integrity, not publisher authenticity.
func resolveObserver(root,platform string)(observerManifest,string,error){
 var m observerManifest
 if root==""||!filepath.IsAbs(root){return m,"",errors.New("请先在设置中选择已安装的 Observer 0.2.0 资源包")}
 root,e:=filepath.EvalSymlinks(root);if e!=nil{return m,"",e}
 b,e:=readKitFile(root,"manifest.json");if e!=nil{return m,"",e}
 if e=json.Unmarshal(b,&m);e!=nil{return m,"",e}
 if m.SchemaVersion!=1||m.Version!="0.2.0"||!hasV1(m.SessionSchemaVersions){return m,"",errors.New("当前仅验证 Observer 正式版 0.2.0 / session v1")}
 target,ok:=m.Platforms[platform]
 if !ok||target.Status!="stable"||target.Executable==""||len(target.SHA256)!=64{return m,"",errors.New("Observer 不包含当前平台的正式二进制")}
 rel:=filepath.FromSlash(target.Executable)
 if filepath.IsAbs(rel)||filepath.VolumeName(rel)!=""||filepath.Clean(rel)!=rel||strings.ContainsAny(target.Executable,"\\:"){return m,"",errors.New("Observer manifest 可执行路径无效")}
 path:=filepath.Join(root,rel)
 back,e:=filepath.Rel(root,path);if e!=nil||back==".."||strings.HasPrefix(back,".."+string(filepath.Separator)){return m,"",errors.New("Observer 可执行路径越界")}
 actual,e:=filepath.EvalSymlinks(path);if e!=nil{return m,"",e}
 if actual!=path{return m,"",errors.New("Observer 可执行路径包含链接")}
 st,e:=os.Lstat(path);if e!=nil{return m,"",e}
 if !st.Mode().IsRegular()||st.Size()>100*1024*1024{return m,"",errors.New("Observer 可执行文件无效")}
 f,e:=os.Open(path);if e!=nil{return m,"",e};defer f.Close()
 b,e=io.ReadAll(io.LimitReader(f,100*1024*1024+1));if e!=nil{return m,"",e}
 if len(b)>100*1024*1024||digest(b)!=strings.ToLower(target.SHA256){return m,"",errors.New("Observer 二进制 SHA256 不匹配")}
 return m,path,nil
}
func ProbeObserver(root string)(ObserverInfo,error){
 platform,backend,e:=observerPlatform();if e!=nil{return ObserverInfo{},e}
 m,path,e:=resolveObserver(root,platform);if e!=nil{return ObserverInfo{},e}
 p:=process(context.Background(),path,[]string{"--capabilities"},filepath.Dir(path),"",nil,5*time.Second)
 if p.State!="ok"{return ObserverInfo{},fmt.Errorf("Observer 能力查询失败: %s %s",p.State,p.Stderr)}
 var c observerCapabilities
 if e=json.Unmarshal([]byte(p.Stdout),&c);e!=nil{return ObserverInfo{},fmt.Errorf("Observer 能力响应无效: %w",e)}
 if e=validateObserverCapabilities(c,m.Version,runtime.GOOS,runtime.GOARCH,backend);e!=nil{return ObserverInfo{},e}
 return ObserverInfo{Version:c.Version,Executable:path,Backend:backend},nil
}

 // Observer v0.2.0 reports implementation IDs (gdb-mi/lldb-dap),
 // whereas session.backend uses selector names (gdb/lldb).
func validateObserverCapabilities(c observerCapabilities,version,platform,arch,backend string)error{
 reported:=map[string]string{"gdb":"gdb-mi","lldb":"lldb-dap"}[backend]
 if reported==""||c.Version!=version||c.Platform!=platform||c.Arch!=arch||!hasV1(c.SessionSchemaVersions)||!hasString(c.Backends,reported)||!hasString(c.SessionFeatures,"backend")||!hasString(c.SessionFeatures,"comparison"){
  return fmt.Errorf("Observer 能力不匹配: version=%q platform=%q arch=%q backends=%v schemas=%v features=%v; 需要 %s/%s %s (%s), session v1",c.Version,c.Platform,c.Arch,c.Backends,c.SessionSchemaVersions,c.SessionFeatures,platform,arch,version,reported)
 }
 return nil
}

type observerSession struct{
 SchemaVersion int `json:"schemaVersion"`
 Backend string `json:"backend"`
 SourcePath string `json:"sourcePath"`
 WorkDir string `json:"workDir"`
 VisibleSourcePaths []string `json:"visibleSourcePaths,omitempty"`
 SkipSourcePaths []string `json:"skipSourcePaths,omitempty"`
 SourceMap map[string]string `json:"sourceMap,omitempty"`
 Compiler string `json:"compiler"`
 CompilerArgs []string `json:"compilerArgs"`
 StdinFile string `json:"stdinFile"`
 ExpectedStdoutFile *string `json:"expectedStdoutFile,omitempty"`
 Comparison string `json:"comparison"`
}
type Observation struct{
 Directory string
 SessionPath string
 PreparedDigest string
}
func(o Observation) Cleanup()error{return os.RemoveAll(o.Directory)}

// Observer validates that mapped user lines equal the frozen original text.
// Restore original comments on user segments, not on the replaced include.
// Insert #line only at segment boundaries (never inside a multiline comment).
func observationSource(s Source,raw []byte,p PreparedSource)([]byte,error){
 if p.KitVersion==""{return raw,nil}
 originals:=sourceLines(raw);masked:=sourceLines(maskC(raw,false));generated:=sourceLines(p.Bytes)
 if len(generated)!=len(p.Origins){return nil,errors.New("观察源码映射无效")}
 var out bytes.Buffer
 prevFile:="";prevLine:=-1
 for i,origin:=range p.Origins{
  line:=generated[i];logical:="preset/generated.h";row:=origin.Line
  if row<1{row=1}
  if origin.File==s.Path{
   if origin.Line<1||origin.Line>len(originals){return nil,errors.New("观察源码行号无效")}
   logical="companion-user.c"
   rawLine:=originals[origin.Line-1]
   if bytes.Equal(line,masked[origin.Line-1]){line=rawLine}else{
    // Only the Kit include itself is replaced by a newline. A comment
    // crossing that boundary cannot be safely restored with #line.
    m:=ppDirective.FindStringSubmatch(strings.TrimRight(string(rawLine),"\r\n"))
    if m==nil||m[1]!="include"||!literalInclude.MatchString(m[2]){return nil,errors.New("观察暂不支持 Exam Kit include 行上的注释；请单独书写该引用")}
   }
  }else{
   part:=strings.Split(origin.File," / ")
   if len(part)==2&&kitID.MatchString(part[1]){logical="preset/"+part[1]+".h"}
  }
  if logical!=prevFile||row!=prevLine+1{fmt.Fprintf(&out,"#line %d %q\n",row,logical)}
  out.Write(line);if !bytes.HasSuffix(line,[]byte("\n")){out.WriteByte('\n')}
  prevFile,prevLine=logical,row
 }
 return out.Bytes(),nil
}

// Build a fresh current-source session; never label it as a historical run.
func PrepareObservation(path,compiler,revision string,selection int,info ObserverInfo)(Observation,error){
 if isCPP(path){return Observation{},errors.New("Observer 0.2.0 仅支持 C99 源码")}
 s,e:=Open(path);if e!=nil{return Observation{},e}
 raw,e:=readSource(s);if e!=nil{return Observation{},e}
 p,e:=PrepareSource(s,raw);if e!=nil{return Observation{},e}
 cs,e:=LoadCases(s.Path);if e!=nil{return Observation{},e}
 if cs.Revision!=revision{return Observation{},errors.New("测试数据已变化，请重新读取")}
 if len(cs.Cases)==0{cs.Cases=[]Case{{Input:""}};selection=0}
 if selection<0||selection>=len(cs.Cases){return Observation{},errors.New("请选择一组测试数据")}
 cc,e:=compilerFor(compiler,false);if e!=nil{return Observation{},e}
 obs,e:=observationSource(s,raw,p);if e!=nil{return Observation{},e}
 dir,e:=os.MkdirTemp("","companion-observe-");if e!=nil{return Observation{},e}
 success:=false;defer func(){if !success{_ = os.RemoveAll(dir)}}()
 write:=func(name string,b []byte)error{return os.WriteFile(filepath.Join(dir,name),b,0600)}
 session:=observerSession{SchemaVersion:1,Backend:info.Backend,SourcePath:"source.c",WorkDir:s.Directory,Compiler:cc,CompilerArgs:[]string{"-lm"},StdinFile:"stdin.txt",Comparison:"trim-line-end"}
 if e=write("source.c",obs);e!=nil{return Observation{},e}
 if p.KitVersion!=""{
  if e=write("user.c",raw);e!=nil{return Observation{},e}
  session.VisibleSourcePaths=[]string{"user.c"};session.SkipSourcePaths=[]string{"source.c"};session.SourceMap=map[string]string{"companion-user.c":"user.c"}
 }
 c:=cs.Cases[selection];if e=write("stdin.txt",[]byte(c.Input));e!=nil{return Observation{},e}
 if c.Expected!=nil{name:="expected.txt";session.ExpectedStdoutFile=&name;if e=write(name,[]byte(*c.Expected));e!=nil{return Observation{},e}}
 b,e:=json.MarshalIndent(session,"","  ");if e!=nil{return Observation{},e}
 if e=write("session.json",b);e!=nil{return Observation{},e}
 success=true
 return Observation{Directory:dir,SessionPath:filepath.Join(dir,"session.json"),PreparedDigest:p.PreparedDigest},nil
}
type ObserverProcess struct{
 cmd *exec.Cmd
 observation Observation
 log *limitedLog
}
type limitedLog struct{b bytes.Buffer}
func(l *limitedLog)Write(p []byte)(int,error){n:=len(p);left:=16384-l.b.Len();if left>0{if len(p)>left{p=p[:left]};_,_=l.b.Write(p)};return n,nil}
func StartObserver(info ObserverInfo,o Observation)(*ObserverProcess,error){
 cmd:=exec.Command(info.Executable,"--session",o.SessionPath)
 cmd.Dir=filepath.Dir(info.Executable)
 log:=&limitedLog{};cmd.Stderr=log
 cmd.WaitDelay=2*time.Second
 if e:=cmd.Start();e!=nil{return nil,e}
 return &ObserverProcess{cmd:cmd,observation:o,log:log},nil
}
func(p *ObserverProcess) Wait()error{
 e:=p.cmd.Wait()
 cleanup:=p.observation.Cleanup()
 if e!=nil{return fmt.Errorf("Observer 启动或退出失败: %w %s",e,p.log.b.String())}
 if cleanup!=nil{return fmt.Errorf("观察会话清理失败: %w",cleanup)}
 return nil
}

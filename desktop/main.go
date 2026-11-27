package main

import (
 "context"
 "embed"
 "errors"
 "fmt"
 "log"
 "os"
 "path/filepath"
 "sync"
 "strings"

 "github.com/Routilet/companion/internal/localrun"
 "github.com/wailsapp/wails/v3/pkg/application"
 "github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

type Desktop struct{
 native *application.App
 mu sync.Mutex
 selected localrun.Source
 running bool
 cancel context.CancelFunc
 done chan struct{}
 root string
 last map[string]localrun.BatchResult
 order []string
 settings localrun.Settings
 settingsPath string
 settingsError string
 progress localrun.Progress
 drop []string
 prepared *localrun.Submission
 kitPackage string
 kitError string
 manual *localrun.Manual
 managerPath string
 observerDone chan struct{}
 observerState string
}
type Navigation struct { Root string `json:"root"`; Source localrun.Source `json:"source"` }
func(d *Desktop) nav()Navigation{return Navigation{d.root,d.selected}}
func(d *Desktop) KitStatus()localrun.KitStatus{
 d.mu.Lock();p,reason:=d.kitPackage,d.kitError;d.mu.Unlock()
 if reason!=""{return localrun.KitStatus{Path:p,Error:reason}}
 k,e:=localrun.LoadKitPackage(p);if e!=nil{return localrun.KitStatus{Path:p,Error:e.Error()}}
 return localrun.KitStatus{Path:p,Version:k.Version}
}
func(d *Desktop) OpenManual()(localrun.Manual,error){
 d.mu.Lock();defer d.mu.Unlock()
 d.manual=nil
 m,e:=localrun.LoadManual(d.selected.Path,d.kitPackage)
 if e==nil{d.manual=&m}
 return m,e
}
func(d *Desktop) SearchManual(query string)([]localrun.ReferenceHit,error){
 d.mu.Lock();defer d.mu.Unlock()
 if d.manual==nil{return nil,errors.New("请先打开函数手册")}
 return localrun.SearchReference(d.manual.Reference,query),nil
}
func(d *Desktop) CopyManualExample(id,key,fingerprint string)error{
 d.mu.Lock()
 if d.manual==nil||d.manual.Fingerprint!=fingerprint{d.mu.Unlock();return errors.New("手册已变化，请重新打开")}
 m,e:=localrun.LoadManual(d.selected.Path,d.kitPackage)
 if e!=nil||m.Fingerprint!=fingerprint||m.Origin!=d.manual.Origin||m.SourcePath!=d.manual.SourcePath{d.mu.Unlock();return errors.New("资源或当前文件已变化，请重新打开手册")}
 example,e:=m.Example(id,key);d.mu.Unlock()
 if e!=nil{return e}
 if !d.native.Clipboard.SetText(example){return errors.New("无法写入系统剪贴板")}
 return nil
}
func(d *Desktop) CopyManualSignature(id,key,fingerprint,signature string)error{
 d.mu.Lock()
 if d.manual==nil||d.manual.Fingerprint!=fingerprint{d.mu.Unlock();return errors.New("手册已变化，请重新打开")}
 m,e:=localrun.LoadManual(d.selected.Path,d.kitPackage)
 if e!=nil||m.Fingerprint!=fingerprint||m.Origin!=d.manual.Origin||m.SourcePath!=d.manual.SourcePath{d.mu.Unlock();return errors.New("资源或当前文件已变化，请重新打开手册")}
 valid:=false
 for _,c:=range m.Reference.Categories{for _,entry:=range c.Entries{if entry.ID!=id{continue};signatures:=entry.Signatures
  if len(entry.Variants)>0{signatures=nil;for _,v:=range entry.Variants{if v.Key==key{signatures=append(append([]string{},entry.CommonSignatures...),v.Signatures...);break}}}
  for _,s:=range signatures{if s==signature{valid=true}}
 }}
 d.mu.Unlock();if !valid{return errors.New("未知函数接口")}
 if !d.native.Clipboard.SetText(signature){return errors.New("无法写入系统剪贴板")};return nil
}
func(d *Desktop) ObserverStatus()localrun.ObserverInfo{
 d.mu.Lock();p:=d.managerPath;d.mu.Unlock()
 info,e:=localrun.ProbeManagedObserver(p);if e!=nil{return localrun.ObserverInfo{Error:e.Error()}}
 return info
}
func(d *Desktop) Observe(compiler,revision string,selection int)error{
 d.mu.Lock();defer d.mu.Unlock()
 if d.running||d.observerDone!=nil||d.selected.Path==""{return errors.New("请先结束当前运行或观察")}
 info,e:=localrun.ProbeManagedObserver(d.managerPath);if e!=nil{return e}
 session,e:=localrun.PrepareObservation(d.selected.Path,compiler,revision,selection,info);if e!=nil{return e}
 p,e:=localrun.StartObserver(info,session);if e!=nil{_ = session.Cleanup();return e}
 done:=make(chan struct{});d.observerDone=done;d.observerState="running"
 go func(){e:=p.Wait();d.mu.Lock();defer d.mu.Unlock()
  if e!=nil{d.observerState=e.Error()}else{d.observerState="closed"}
  d.observerDone=nil;close(done)
 }()
 return nil
}
func(d *Desktop) ObservationState()string{d.mu.Lock();defer d.mu.Unlock();return d.observerState}
func(d *Desktop) Config()(localrun.Settings,error){d.mu.Lock();defer d.mu.Unlock();return d.settings,nil}
func(d *Desktop) ConfigWarning()string{d.mu.Lock();defer d.mu.Unlock();return d.settingsError}
func(d *Desktop) StoreConfig(c,cpp,editor string)error{
 d.mu.Lock();defer d.mu.Unlock()
 if d.settingsPath==""{return errors.New("无法定位应用配置目录")}
 s:=localrun.Settings{Root:d.root,C:c,CPP:cpp,Editor:editor}
 if d.selected.Path!=""{if r,e:=filepath.Rel(d.root,d.selected.Path);e==nil{s.Selected=r}}
 if e:=localrun.SaveSettings(d.settingsPath,s);e!=nil{return e}
 d.settings=s;d.settingsError="";return nil
}
func(d *Desktop) TakeDrop()[]string{d.mu.Lock();defer d.mu.Unlock();p:=d.drop;d.drop=nil;return p}
func(d *Desktop) OpenDropped(path string)(Navigation,error){
 d.mu.Lock();defer d.mu.Unlock()
 if d.running{return Navigation{},errors.New("正在运行")}
 if s,e:=localrun.Open(path);e==nil{
  root,e:=localrun.OpenDirectory(s.Directory);if e!=nil{return Navigation{},e}
  s,e=localrun.Open(filepath.Join(root,s.Name));if e!=nil{return Navigation{},e}
  d.root=root;d.selected=s;d.prepared=nil;return d.nav(),nil
 }
 root,e:=localrun.OpenDirectory(path);if e!=nil{return Navigation{},errors.New("请拖入一个 C/C++ 文件或普通文件夹")}
 d.root=root;d.selected=localrun.Source{};d.prepared=nil;return d.nav(),nil
}
func(d *Desktop) DetectEditors()[]localrun.EditorOption{return localrun.DetectEditors()}
func(d *Desktop) ChooseEditor()(string,error){
 return d.native.Dialog.OpenFile().SetTitle("选择编辑器可执行文件").PromptForSingleSelection()
}
func(d *Desktop) OpenEditor(editor string)error{
 d.mu.Lock();p:=d.selected.Path;d.mu.Unlock()
 return localrun.OpenEditor(p,editor)
}
func(d *Desktop) CreateFolder(dir,name string)(string,error){
 d.mu.Lock();defer d.mu.Unlock();if d.running{return "",errors.New("正在运行")};return localrun.CreateFolder(d.root,dir,name)
}
func(d *Desktop) RevealFolder(path string)error{
 d.mu.Lock();root:=d.root;d.mu.Unlock()
 p,e:=localrun.Within(root,path);if e!=nil{return e};return localrun.RevealDirectory(p)
}
func(d *Desktop) Reveal()error{
 d.mu.Lock();p:=d.root;if d.selected.Path!=""{p=d.selected.Directory};d.mu.Unlock()
 if p==""{return errors.New("未选择目录")};return localrun.RevealDirectory(p)
}
func(d *Desktop) Prepare(compiler string)(localrun.Submission,error){
 d.mu.Lock();defer d.mu.Unlock()
 if d.running||d.selected.Path==""{return localrun.Submission{},errors.New("当前不能复制")}
 var last *localrun.BatchResult
 if r,ok:=d.last[d.selected.Path];ok{last=&r}
 s,e:=localrun.PrepareSubmission(d.selected.Path,compiler,last);if e==nil{d.prepared=&s};return s,e
}
func(d *Desktop) CopyPrepared(compiler,digest string)error{
 d.mu.Lock()
 if d.running||d.prepared==nil||d.prepared.Source.Path!=d.selected.Path||d.prepared.Digest!=digest{d.mu.Unlock();return errors.New("预览已失效，请重新预览")}
 s,e:=localrun.CheckSubmission(*d.prepared,compiler);d.prepared=nil;d.mu.Unlock()
 if e!=nil{return e}
 if !d.native.Clipboard.SetText(s.Text){return errors.New("无法写入系统剪贴板")}
 return nil
}
func(d *Desktop) Progress()localrun.Progress{d.mu.Lock();defer d.mu.Unlock();return d.progress}

func(d *Desktop) Version()string{return buildVersion}
func(d *Desktop) Initial()(Navigation,error){d.mu.Lock();defer d.mu.Unlock();return d.nav(),nil}
func(d *Desktop) ChooseSource()(Navigation,error){
 d.mu.Lock();if d.running{d.mu.Unlock();return Navigation{},errors.New("请先结束当前运行")};d.mu.Unlock()
 p,e:=d.native.Dialog.OpenFile().SetTitle("选择 C/C++ 源文件").AddFilter("C/C++ 源文件","*.c;*.cpp;*.cc;*.cxx").PromptForSingleSelection()
 if e!=nil||p==""{return Navigation{},e}
 s,e:=localrun.Open(p);if e!=nil{return Navigation{},e}
 root,e:=localrun.OpenDirectory(s.Directory);if e!=nil{return Navigation{},e}
 s,e=localrun.Open(filepath.Join(root,s.Name));if e!=nil{return Navigation{},e}
 d.mu.Lock();defer d.mu.Unlock()
 if d.running{return Navigation{},errors.New("正在运行")}
 d.root=root;d.selected=s;return d.nav(),nil
}
func(d *Desktop) ChooseFolder()(Navigation,error){
 p,e:=d.native.Dialog.OpenFile().SetTitle("打开文件夹").CanChooseFiles(false).CanChooseDirectories(true).PromptForSingleSelection()
 if e!=nil||p==""{return Navigation{},e}
 root,e:=localrun.OpenDirectory(p);if e!=nil{return Navigation{},e}
 d.mu.Lock();defer d.mu.Unlock()
 if d.running{return Navigation{},errors.New("正在运行")}
 d.root=root;d.selected=localrun.Source{};return d.nav(),nil
}
func(d *Desktop) List(dir string)([]localrun.Entry,error){
 d.mu.Lock();root:=d.root;d.mu.Unlock();return localrun.ListDirectory(root,dir)
}
func(d *Desktop) Select(path string)(Navigation,error){
 d.mu.Lock();defer d.mu.Unlock()
 if d.running{return Navigation{},errors.New("正在运行")}
 p,e:=localrun.Within(d.root,path);if e!=nil{return Navigation{},e}
 s,e:=localrun.Open(p);if e!=nil{return Navigation{},e}
 d.selected=s;return d.nav(),nil
}
// ChooseCreateDirectory is read-only; cancelling or browsing does not change navigation.
func(d *Desktop) ChooseCreateDirectory()(string,error){
 p,e:=d.native.Dialog.OpenFile().SetTitle("选择新建文件的位置").CanChooseFiles(false).CanChooseDirectories(true).PromptForSingleSelection()
 if e!=nil||p==""{return "",e}
 return localrun.OpenDirectory(p)
}
func(d *Desktop) Create(dir,name string)(Navigation,error){
 d.mu.Lock();defer d.mu.Unlock()
 if d.running{return Navigation{},errors.New("正在运行")}
 target,e:=filepath.Abs(dir);if e!=nil{return Navigation{},e}
 root:=d.root
 if root==""{root,e=localrun.OpenDirectory(target);if e!=nil{return Navigation{},e};target=root}else if _,e=localrun.Within(root,target);e!=nil{
  rel,relErr:=filepath.Rel(root,target)
  // Only explicit external directories may establish a new root. Do not
  // reinterpret a rejected descendant (hidden, symlink, etc.) as external.
  if relErr==nil&&rel!=".."&&!filepath.IsAbs(rel)&&!strings.HasPrefix(rel,".."+string(filepath.Separator)){return Navigation{},e}
  root,e=localrun.OpenDirectory(target);if e!=nil{return Navigation{},e};target=root
 }
 s,e:=localrun.CreateSourceWithKit(root,target,name,d.kitPackage);if e!=nil{return Navigation{},e}
 d.root=root;d.selected=s;d.prepared=nil;return d.nav(),nil
}
func(d *Desktop) Rename(name string)(Navigation,error){
 d.mu.Lock();defer d.mu.Unlock()
 if d.running||d.selected.Path==""{return Navigation{},errors.New("当前不能重命名")}
 old:=d.selected.Path
 s,e:=localrun.RenameSource(d.root,old,name);if e!=nil{return Navigation{},e}
 delete(d.last,old);delete(d.last,s.Path);d.selected=s;return d.nav(),nil
}
func(d *Desktop) Orphans()([]localrun.Orphan,error){
 d.mu.Lock();defer d.mu.Unlock()
 return localrun.ListOrphans(d.selected.Path)
}
func(d *Desktop) Restore(name,rev string)(localrun.CaseSet,error){
 d.mu.Lock();defer d.mu.Unlock()
 if d.running{return localrun.CaseSet{},errors.New("正在运行")}
 return localrun.RestoreOrphan(d.selected.Path,name,rev)
}
func(d *Desktop) Import(rev string,appendExisting bool)(*localrun.CaseSet,error){
 d.mu.Lock();selected:=d.selected.Path;d.mu.Unlock()
 p,e:=d.native.Dialog.OpenFile().SetTitle("导入测试数据").AddFilter("JSON","*.json").ShowHiddenFiles(true).PromptForSingleSelection()
 if e!=nil||p==""{return nil,e}
 d.mu.Lock();defer d.mu.Unlock()
 if d.running||selected==""||selected!=d.selected.Path{return nil,errors.New("文件已切换")}
 cs,e:=localrun.ImportCases(selected,p,rev,appendExisting);return &cs,e
}
func(d *Desktop) Cached(compiler string)(*localrun.BatchResult,error){
 d.mu.Lock();defer d.mu.Unlock()
 r,ok:=d.last[d.selected.Path];if !ok{return nil,nil}
 r.Stale=r.Stale||!localrun.Current(r,compiler)
 return &r,nil
}
func(d *Desktop) remember(r localrun.BatchResult){
 if d.last==nil{d.last=map[string]localrun.BatchResult{}}
 path:=r.Source.Path
 for i,p:=range d.order{if p==path{d.order=append(d.order[:i],d.order[i+1:]...);break}}
 d.order=append(d.order,path);d.last[path]=r
 // Keep only a small session cache; a result may contain large outputs.
 for len(d.order)>4{delete(d.last,d.order[0]);d.order=d.order[1:]}
}
// CheckSelected validates the current path even if its parent is collapsed.
func(d *Desktop) CheckSelected()(bool,error){
 d.mu.Lock();defer d.mu.Unlock()
 if d.selected.Path==""{return false,nil}
 _,e:=localrun.Within(d.root,d.selected.Path)
 if e==nil{_,e=localrun.Open(d.selected.Path)}
 if e!=nil{delete(d.last,d.selected.Path);return false,nil}
 return true,nil
}
func(d *Desktop) Invalidate(path string){
 d.mu.Lock();defer d.mu.Unlock();delete(d.last,path)
}
func(d *Desktop) LoadCases()(localrun.CaseSet,error){
 d.mu.Lock();defer d.mu.Unlock()
 if d.selected.Path==""{return localrun.CaseSet{},errors.New("请选择源文件")}
 return localrun.LoadCases(d.selected.Path)
}
func(d *Desktop) SaveCases(cs []localrun.Case,rev string)(localrun.CaseSet,error){
 d.mu.Lock();defer d.mu.Unlock()
 if d.running||d.selected.Path==""{return localrun.CaseSet{},errors.New("当前不能保存")}
 return localrun.SaveCases(d.selected.Path,cs,rev)
}
func(d *Desktop) RunBatch(compiler,revision string,selection int)(localrun.BatchResult,error){
 d.mu.Lock()
 if d.running||d.selected.Path==""{d.mu.Unlock();return localrun.BatchResult{},errors.New("请选择源文件并结束之前的运行")}
 ctx,cancel:=context.WithCancel(context.Background())
 path:=d.selected.Path;d.running=true;d.cancel=cancel;d.done=make(chan struct{});d.progress=localrun.Progress{Phase:"compiling"};d.mu.Unlock()
 defer func(){cancel();d.mu.Lock();d.running=false;d.progress=localrun.Progress{};d.cancel=nil;close(d.done);d.done=nil;d.mu.Unlock()}()
 r,e:=localrun.RunBatchProgress(ctx,path,compiler,revision,selection,func(p localrun.Progress){d.mu.Lock();d.progress=p;d.mu.Unlock()})
 if e==nil{d.mu.Lock();d.remember(r);d.mu.Unlock()}
 return r,e
}
func(d *Desktop) IsCurrent(compiler string)(bool,error){
 d.mu.Lock();defer d.mu.Unlock()
 r,ok:=d.last[d.selected.Path];if !ok{return false,nil}
 return !r.Stale&&localrun.Current(r,compiler),nil
}
func(d *Desktop) Cancel(){d.mu.Lock();defer d.mu.Unlock();if d.cancel!=nil{d.cancel()}}
func(d *Desktop) shutdown(){
 d.mu.Lock();if d.cancel!=nil{d.cancel()};done:=d.done;observerDone:=d.observerDone;d.mu.Unlock()
 if done!=nil{<-done}
 // Keep session files alive until Observer exits normally. Never kill its GUI
 // while a debugger may need to clean up the inferior.
 if observerDone!=nil{<-observerDone}
}
var buildVersion="0.1.0-beta.2"
func main(){
 if len(os.Args)==2&&os.Args[1]=="--version"{fmt.Println(buildVersion);return}
 if len(os.Args)==2&&os.Args[1]=="--check-install"{p,e:=localrun.ResolveInstalledKit();if e!=nil{fmt.Fprintln(os.Stderr,e);os.Exit(1)};k,e:=localrun.LoadKitPackage(p);if e!=nil{fmt.Fprintln(os.Stderr,e);os.Exit(1)};fmt.Println("companion "+buildVersion+" exam-kit "+k.Version);return}
 kit,e:=localrun.ResolveInstalledKit()
 d:=&Desktop{kitPackage:kit,managerPath:os.Getenv(localrun.RoutiletExecutableEnv)}
 if e!=nil{d.kitError=e.Error()}
 if kit==""&&os.Getenv(localrun.KitPreviewEnv)!=""{d.kitPackage=os.Getenv(localrun.KitPreviewEnv);d.kitError=""}
 if path,e:=localrun.SettingsPath();e==nil{d.settingsPath=path;if s,e:=localrun.LoadSettings(path);e==nil{d.settings=s}else{d.settingsError=e.Error()}}else{d.settingsError=e.Error()}
 if len(os.Args)==1&&d.settings.Root!=""{
  if root,e:=localrun.OpenDirectory(d.settings.Root);e==nil{d.root=root;if d.settings.Selected!=""&&!filepath.IsAbs(d.settings.Selected){if p,e:=localrun.Within(root,filepath.Join(root,d.settings.Selected));e==nil{d.selected,_=localrun.Open(p)}}}
 }
 if len(os.Args)==2{if s,e:=localrun.Open(os.Args[1]);e==nil{if root,e:=localrun.OpenDirectory(s.Directory);e==nil{d.root=root;d.selected,_=localrun.Open(filepath.Join(root,s.Name))}}else if root,e:=localrun.OpenDirectory(os.Args[1]);e==nil{d.root=root}else{fmt.Fprintln(os.Stderr,e)}}
 app:=application.New(application.Options{
  Name:"Companion",Description:"Local C programming companion",
  Services:[]application.Service{application.NewService(d)},
  Assets:application.AssetOptions{Handler:application.BundledAssetFileServer(assets)},
  Mac:application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed:true},
  OnShutdown:d.shutdown,
 })
 if icon,e:=assets.ReadFile("frontend/dist/icons/companion.png");e==nil{app.SetIcon(icon)}
 d.native=app
 win:=app.Window.NewWithOptions(application.WebviewWindowOptions{Title:"Companion",Width:1000,Height:700,URL:"/",EnableFileDrop:true})
 win.OnWindowEvent(events.Common.WindowFilesDropped,func(e *application.WindowEvent){if ctx:=e.Context();ctx!=nil{d.mu.Lock();d.drop=append([]string(nil),ctx.DroppedFiles()...);d.mu.Unlock()}})
 if e:=app.Run();e!=nil{log.Fatal(e)}
}

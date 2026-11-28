package localrun

import(
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "os"
 "path/filepath"
 "runtime"
 "strings"
 "time"
)

const RoutiletCallProtocol="routilet-app-call/1"
func ProbeManagedObserver(manager string)(ObserverInfo,error){
 if manager==""||!filepath.IsAbs(manager){return ObserverInfo{},errors.New("请通过支持跨应用调用的 Routilet 启动 Companion")}
 st,e:=os.Stat(manager);if e!=nil||!st.Mode().IsRegular(){return ObserverInfo{},errors.New("Routilet 调用入口不可用")}
 _,backend,e:=observerPlatform();if e!=nil{return ObserverInfo{},e}
 probe:=process(context.Background(),manager,[]string{"--app-call-probe"},filepath.Dir(manager),"",nil,5*time.Second)
 if probe.State!="ok"||strings.TrimSpace(probe.Stdout)!=RoutiletCallProtocol{return ObserverInfo{},errors.New("Routilet 版本不支持应用调用协议 v1，请更新后重新启动 Companion")}
 p:=process(context.Background(),manager,[]string{"--query-app","observer","--","--capabilities"},filepath.Dir(manager),"",nil,15*time.Second)
 if p.State!="ok"{return ObserverInfo{},fmt.Errorf("Observer 不可用，请在 Routilet 安装或检查版本: %s %s",p.State,p.Stderr)}
 var c observerCapabilities
 if e=json.Unmarshal([]byte(p.Stdout),&c);e!=nil{return ObserverInfo{},fmt.Errorf("Observer 能力响应无效: %w",e)}
 if e=validateObserverCapabilities(c,"",runtime.GOOS,runtime.GOARCH,backend);e!=nil{return ObserverInfo{},e}
 return ObserverInfo{Version:c.Version,Executable:manager,Backend:backend,Managed:true},nil
}

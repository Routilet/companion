package ojadapter

import (
 "bytes"
 "context"
 "errors"
 "os"
 "os/exec"
 "path/filepath"
 "strings"
 "time"
)

const managerProtocol="routilet-adapter-call/1"
const managerProbeTimeout=3*time.Second

// Limit the untrusted output; never include child output in account errors.
type probeOutput struct{ b bytes.Buffer; overflow bool }
func (w *probeOutput) Write(p []byte)(int,error){
 n:=len(p);left:=1024-w.b.Len();if len(p)>left{w.overflow=true;p=p[:left]}
 _,_=w.b.Write(p);return n,nil
}
func managerProbe(parent context.Context,path string,args ...string)(string,error){
 ctx,cancel:=context.WithTimeout(parent,managerProbeTimeout);defer cancel()
 cmd:=exec.CommandContext(ctx,path,args...);cmd.Dir=filepath.Dir(path);configure(cmd)
 var out probeOutput;cmd.Stdout=&out;cmd.Stderr=nil;cmd.WaitDelay=time.Second
 if e:=cmd.Run();e!=nil||out.overflow{return "",errors.New("manager probe failed")}
 return strings.TrimSpace(out.b.String()),nil
}
func checkManaged(ctx context.Context,path string)error{
 if !filepath.IsAbs(path){return errors.New("Routilet 路径无效，请从新版 Routilet 打开 Companion")}
 if st,e:=os.Stat(path);e!=nil||!st.Mode().IsRegular(){return errors.New("找不到 Routilet，请从新版 Routilet 打开 Companion")}
 if protocol,e:=managerProbe(ctx,path,"--adapter-call-probe");e!=nil||protocol!=managerProtocol{return errors.New("当前 Routilet 不支持 Adapter 调用，请更新 Routilet 后重新打开 Companion")}
 if _,e:=managerProbe(ctx,path,"--adapter-available","accoding-adapter");e!=nil{return errors.New("Adapter 未安装、已禁用或校验失败；请在 Routilet 检查（测试版需测试通道）")}
 return nil
}

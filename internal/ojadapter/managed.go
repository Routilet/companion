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
const managerProbeTimeout=10*time.Second

// Bound stdout and stderr; never return arbitrary child output or filesystem paths.
type probeOutput struct{ b bytes.Buffer; overflow bool }
func (w *probeOutput) Write(p []byte)(int,error){
 n:=len(p);left:=8192-w.b.Len();if len(p)>left{w.overflow=true;p=p[:left]}
 _,_=w.b.Write(p);return n,nil
}
var errManagerBusy=errors.New("Routilet 正在安装、更新或清理组件，请稍后重新检查")
func classifyManagerFailure(stderr string)error{
 switch {
 case strings.Contains(stderr,"另一 Routilet 实例正在操作"):return errManagerBusy
 case strings.Contains(stderr,"应用已禁用"):return errors.New("Accoding Adapter 已禁用，请在 Routilet 中启用")
 case strings.Contains(stderr,"activation.json")&&(strings.Contains(stderr,"no such file")||strings.Contains(stderr,"cannot find")||strings.Contains(stderr,"找不到")):return errors.New("Accoding Adapter 未安装，请在 Routilet 测试通道安装")
 case strings.Contains(stderr,"安装记录无效"):return errors.New("Accoding Adapter 安装记录无效，请在 Routilet 修复")
 case strings.Contains(stderr,"原生文件校验失败"):return errors.New("Accoding Adapter 文件 SHA256 校验失败，请在 Routilet 修复")
 case strings.Contains(stderr,"适配器身份或协议不兼容")||strings.Contains(stderr,"应用清单不兼容")||strings.Contains(stderr,"当前平台没有有效入口"):return errors.New("Accoding Adapter 安装清单、平台或协议不兼容，请更新或修复")
 case strings.Contains(stderr,"解析安装目录")||strings.Contains(stderr,"解析软件包目录")||strings.Contains(stderr,"读取应用清单")||strings.Contains(stderr,"解析原生入口")||strings.Contains(stderr,"入口不是普通文件")||strings.Contains(stderr,"路径超出"):return errors.New("Accoding Adapter 安装文件缺失或路径异常，请在 Routilet 修复")
 default:return errors.New("Routilet 离线检查失败（未识别错误），请确认已安装并启用 Adapter 后重新检查")
 }
}
func managerProbe(parent context.Context,path string,args ...string)(string,error){
 ctx,cancel:=context.WithTimeout(parent,managerProbeTimeout);defer cancel()
 cmd:=exec.CommandContext(ctx,path,args...);cmd.Dir=filepath.Dir(path);configure(cmd)
 var out,stderr probeOutput;cmd.Stdout=&out;cmd.Stderr=&stderr;cmd.WaitDelay=time.Second
 e:=cmd.Run()
 if ctx.Err()!=nil{return "",errors.New("Routilet 离线检查超时或已取消，请重新检查")}
 if out.overflow||stderr.overflow{return "",errors.New("Routilet 检查输出异常，请更新 Routilet")}
 if e!=nil{
  var exit *exec.ExitError
  if !errors.As(e,&exit){return "",errors.New("无法启动 Routilet 检查进程，请重新打开 Companion")}
  return "",classifyManagerFailure(stderr.b.String())
 }
 return strings.TrimSpace(out.b.String()),nil
}
func checkManaged(ctx context.Context,path string)error{
 if !filepath.IsAbs(path){return errors.New("Routilet 路径无效，请从新版 Routilet 打开 Companion")}
 if st,e:=os.Stat(path);e!=nil||!st.Mode().IsRegular(){return errors.New("找不到 Routilet，请从新版 Routilet 打开 Companion")}
 protocol,e:=managerProbe(ctx,path,"--adapter-call-probe")
 if e!=nil{return e}
 if protocol!=managerProtocol{return errors.New("当前 Routilet 不支持 Adapter 调用，请更新 Routilet 后重新打开 Companion")}
 for attempt:=0;attempt<3;attempt++{
  _,e=managerProbe(ctx,path,"--adapter-available","accoding-adapter")
  if !errors.Is(e,errManagerBusy)||attempt==2{return e}
  select{case <-ctx.Done():return ctx.Err();case <-time.After(300*time.Millisecond):}
 }
 return e
}

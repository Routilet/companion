package localrun
import("testing";"time")
func TestKitLaunchLockRetry(t *testing.T){
 calls:=0;waits:=0;p:=retryKitExport(func()Process{calls++;if calls<3{return Process{State:"failed",Stderr:"另一 Routilet 实例正在操作，请稍后重试"}};return Process{State:"ok",Stdout:"bundle"}},func(time.Duration){waits++});if p.State!="ok"||calls!=3||waits!=2{t.Fatal(p,calls,waits)}
 calls=0;p=retryKitExport(func()Process{calls++;return Process{State:"failed",Stderr:"资源完整性校验失败"}},func(time.Duration){t.Fatal("retried integrity failure")});if calls!=1||p.State!="failed"{t.Fatal(p,calls)}
 calls=0;p=retryKitExport(func()Process{calls++;return Process{State:"failed",Stderr:"另一 Routilet 实例正在操作"}},func(time.Duration){});if calls!=8||p.State!="failed"{t.Fatal(p,calls)}
}

//go:build !windows

package localrun

import (
 "os/exec"
 "syscall"
)
type processOwner struct{stop,close func()}
func configureProcess(cmd *exec.Cmd){cmd.SysProcAttr=&syscall.SysProcAttr{Setpgid:true}}
func ownProcess(cmd *exec.Cmd)processOwner{
 return processOwner{stop:func(){_ = syscall.Kill(-cmd.Process.Pid,syscall.SIGKILL)},close:func(){}}
}

//go:build windows

package localrun

import (
 "os/exec"
 "strconv"
 "syscall"
 "unsafe"
 "golang.org/x/sys/windows"
)
type processOwner struct{stop,close func()}
func configureProcess(cmd *exec.Cmd){cmd.SysProcAttr=&syscall.SysProcAttr{HideWindow:true,CreationFlags:0x08000000}}
func ownProcess(cmd *exec.Cmd)processOwner{
 fallback:=processOwner{stop:func(){
  // A completed leader cannot reliably be used to discover descendants.
  // This fallback is best-effort when the host disallows nested jobs.
  killer:=exec.Command("taskkill","/PID",strconv.Itoa(cmd.Process.Pid),"/T","/F")
  configureProcess(killer);_ = killer.Run();_ = cmd.Process.Kill()
 },close:func(){}}
 job,e:=windows.CreateJobObject(nil,nil);if e!=nil{return fallback}
 info:=windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
 info.BasicLimitInformation.LimitFlags=windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
 if _,e=windows.SetInformationJobObject(job,windows.JobObjectExtendedLimitInformation,uintptr(unsafe.Pointer(&info)),uint32(unsafe.Sizeof(info)));e!=nil{windows.CloseHandle(job);return fallback}
 proc,e:=windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE,false,uint32(cmd.Process.Pid))
 if e!=nil{windows.CloseHandle(job);return fallback}
 e=windows.AssignProcessToJobObject(job,proc);windows.CloseHandle(proc)
 if e!=nil{windows.CloseHandle(job);return fallback}
 return processOwner{stop:func(){_ = windows.TerminateJobObject(job,1)},close:func(){_ = windows.CloseHandle(job)}}
}

//go:build !windows

package localrun

import (
 "os/exec"
 "runtime"
)
func launchDefault(path string,directory bool)error{
 var cmd *exec.Cmd
 if runtime.GOOS=="darwin"{if directory{cmd=exec.Command("open",path)}else{cmd=exec.Command("open","-t",path)}}else{cmd=exec.Command("xdg-open",path)}
 if e:=cmd.Start();e!=nil{return e};go func(){_ = cmd.Wait()}();return nil
}

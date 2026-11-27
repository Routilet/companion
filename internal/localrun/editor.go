package localrun

import (
 "os/exec"
 "path/filepath"
)
func OpenEditor(path,editor string)error{
 s,e:=Open(path);if e!=nil{return e}
 if editor==""{return launchDefault(s.Path,false)}
 p,e:=exec.LookPath(editor);if e!=nil{return e}
 p,e=filepath.Abs(p);if e!=nil{return e}
 cmd:=exec.Command(p,s.Path);cmd.Dir=s.Directory
 if e=cmd.Start();e!=nil{return e};go func(){_ = cmd.Wait()}();return nil
}
func RevealDirectory(path string)error{
 p,e:=OpenDirectory(path);if e!=nil{return e};return launchDefault(p,true)
}

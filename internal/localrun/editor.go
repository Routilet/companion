package localrun

import (
 "os/exec"
 "errors"
 "os"
 "runtime"
 "strings"
 "path/filepath"
)
func OpenEditor(path,editor string)error{
 s,e:=Open(path);if e!=nil{return e}
 if editor==SystemEditor{return launchDefault(s.Path,false)}
 if editor==""{return errors.New("请先选择代码编辑器")}
 if runtime.GOOS=="darwin"&&strings.HasSuffix(strings.ToLower(editor),".app"){if st,e:=os.Stat(editor);e!=nil||!st.IsDir(){return errors.New("编辑器应用不存在")};cmd:=exec.Command("open","-a",editor,s.Path);return cmd.Run()}
 p,e:=exec.LookPath(editor);if e!=nil{return e}
 p,e=filepath.Abs(p);if e!=nil{return e}
 cmd:=exec.Command(p,s.Path);cmd.Dir=s.Directory
 if e=cmd.Start();e!=nil{return e};go func(){_ = cmd.Wait()}();return nil
}
func RevealDirectory(path string)error{
 p,e:=OpenDirectory(path);if e!=nil{return e};return launchDefault(p,true)
}

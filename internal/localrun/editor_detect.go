package localrun

import (
 "os"
 "os/exec"
 "path/filepath"
 "runtime"
 "strings"
)

// SystemEditor is an explicit opt-in, not an implicit fallback.
const SystemEditor = "@system"
type EditorOption struct {
 Name string `json:"name"`
 Path string `json:"path"`
}
type editorSpec struct { name, command string; paths []string }
func DetectEditors() []EditorOption {
 home,_:=os.UserHomeDir()
 var specs []editorSpec
 if runtime.GOOS=="windows" {
  local,program,program86:=os.Getenv("LOCALAPPDATA"),os.Getenv("ProgramFiles"),os.Getenv("ProgramFiles(x86)")
  specs=[]editorSpec{
   {"Visual Studio Code","code",[]string{filepath.Join(local,"Programs","Microsoft VS Code","Code.exe"),filepath.Join(program,"Microsoft VS Code","Code.exe")}},
   {"Dev-C++","devcpp",[]string{filepath.Join(program86,"Dev-Cpp","devcpp.exe"),filepath.Join(program,"Dev-Cpp","devcpp.exe"),filepath.Join(program86,"Embarcadero","Dev-Cpp","devcpp.exe")}},
   {"Code::Blocks","codeblocks",[]string{filepath.Join(program,"CodeBlocks","codeblocks.exe"),filepath.Join(program86,"CodeBlocks","codeblocks.exe")}},
   {"Notepad++","notepad++",[]string{filepath.Join(program,"Notepad++","notepad++.exe"),filepath.Join(program86,"Notepad++","notepad++.exe")}},
   {"Cursor","cursor",[]string{filepath.Join(local,"Programs","cursor","Cursor.exe")}},
   {"Sublime Text","subl",[]string{filepath.Join(program,"Sublime Text","sublime_text.exe")}},
  }
 }else if runtime.GOOS=="darwin"{
  for _,s:=range []struct{name,app,command string}{{"Visual Studio Code","Visual Studio Code.app","code"},{"CLion","CLion.app","clion"},{"Xcode","Xcode.app",""},{"Zed","Zed.app","zed"},{"Sublime Text","Sublime Text.app","subl"},{"Cursor","Cursor.app","cursor"}}{
   specs=append(specs,editorSpec{s.name,s.command,[]string{filepath.Join("/Applications",s.app),filepath.Join(home,"Applications",s.app)}})
  }
 }else{
  specs=[]editorSpec{{"Visual Studio Code","code",nil},{"Sublime Text","subl",nil}}
 }
 return detectEditorSpecs(specs)
}
func detectEditorSpecs(specs []editorSpec) []EditorOption{
 out:=[]EditorOption{}
 for _,s:=range specs{
  candidates:=append([]string(nil),s.paths...)
  if s.command!=""{if p,e:=exec.LookPath(s.command);e==nil{candidates=append(candidates,p)}}
  for _,p:=range candidates{
   if !filepath.IsAbs(p){continue}
   st,e:=os.Stat(p);if e!=nil{continue}
   if !(st.Mode().IsRegular()||(runtime.GOOS=="darwin"&&st.IsDir()&&strings.HasSuffix(strings.ToLower(p),".app"))){continue}
   out=append(out,EditorOption{s.name,p});break
  }
 }
 return out
}

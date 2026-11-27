package localrun

import (
 "encoding/json"
 "errors"
 "fmt"
 "os"
 "path/filepath"
)

const PackageName="@routilet/companion"
const RoutiletExecutableEnv="ROUTILET_EXECUTABLE"

// Only locate the package containing this executable. Do not search cwd,
// arbitrary ancestors or global node_modules. The executable lives in bin/
// win32-x64 or bin/darwin-arm64/Companion.app/Contents/MacOS.
func InstalledKit(exe string)(string,error){
 exe,e:=filepath.EvalSymlinks(exe);if e!=nil{return "",e}
 dir:=filepath.Dir(exe)
 for i:=0;i<7;i++{
  b,e:=readKitFile(dir,"package.json")
  if e==nil{
   var pkg struct{Name string;Dependencies map[string]string}
   if e=json.Unmarshal(b,&pkg);e!=nil{return "",e}
   if pkg.Name==PackageName{
    required:=pkg.Dependencies[KitPackageName]
    if !kitVersion.MatchString(required){return "",errors.New("Companion 未声明固定 Exam Kit 版本")}
    // npm: nested dependency or hoisted sibling; pnpm: sibling within
    // this exact virtual-store package's node_modules.
    candidates:=[]string{filepath.Join(dir,"node_modules",filepath.FromSlash(KitPackageName))}
    scope:=filepath.Dir(dir);modules:=filepath.Dir(scope)
    if filepath.Base(scope)=="@routilet"&&filepath.Base(modules)=="node_modules"{
     candidates=append(candidates,filepath.Join(modules,filepath.FromSlash(KitPackageName)))
    }
    for _,p:=range candidates{
     if _,e=os.Stat(p);os.IsNotExist(e){continue}
     k,e:=LoadKitPackage(p);if e!=nil{return "",fmt.Errorf("Exam Kit 安装异常: %w",e)}
     if k.Version!=required{return "",fmt.Errorf("Exam Kit 版本不匹配: %s / %s",k.Version,required)}
     return filepath.EvalSymlinks(p)
    }
    return "",errors.New("Exam Kit 未安装，请通过 Routilet 修复 Companion")
   }
   return "",errors.New("可执行文件不属于 Companion 发行包")
  }else if !os.IsNotExist(e){return "",e}
  p:=filepath.Dir(dir);if p==dir{break};dir=p
 }
 return "",errors.New("未找到 Companion 安装清单；请通过 Routilet 安装")
}

func ResolveInstalledKit()(string,error){
 exe,e:=os.Executable();if e!=nil{return "",e}
 return InstalledKit(exe)
}

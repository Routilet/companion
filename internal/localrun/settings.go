package localrun

import (
 "encoding/json"
 "errors"
 "io"
 "os"
 "path/filepath"
 "runtime"
)

type Settings struct{
 Root string `json:"root"`
 Selected string `json:"selected"`
 C string `json:"c"`
 CPP string `json:"cpp"`
 Editor string `json:"editor"`
}
func SettingsPath()(string,error){
 base,e:=os.UserConfigDir();if e!=nil{return "",e}
 if runtime.GOOS=="windows"{if p:=os.Getenv("LOCALAPPDATA");p!=""{base=p}}
 return filepath.Join(base,"Companion","settings.json"),nil
}
func LoadSettings(path string)(Settings,error){
 f,e:=os.Open(path);if errors.Is(e,os.ErrNotExist){return Settings{},nil};if e!=nil{return Settings{},e};defer f.Close()
 b,e:=io.ReadAll(io.LimitReader(f,65537));if e!=nil{return Settings{},e}
 if len(b)>65536{return Settings{},errors.New("配置文件过大")}
 var s Settings;if e=json.Unmarshal(b,&s);e!=nil{return Settings{},e}
 return s,nil
}
func SaveSettings(path string,s Settings)error{
 b,e:=json.MarshalIndent(s,"","  ");if e!=nil{return e}
 if len(b)>65536{return errors.New("配置文件过大")}
 dir:=filepath.Dir(path);if e=os.MkdirAll(dir,0700);e!=nil{return e}
 f,e:=os.CreateTemp(dir,".settings-*");if e!=nil{return e};defer os.Remove(f.Name())
 if _,e=f.Write(b);e!=nil{f.Close();return e}
 if e=f.Sync();e!=nil{f.Close();return e}
 if e=f.Close();e!=nil{return e}
 return replaceCases(f.Name(),path)
}

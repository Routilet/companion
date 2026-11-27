//go:build windows

package localrun

import "golang.org/x/sys/windows"

func launchDefault(path string,directory bool)error{
 verb,e:=windows.UTF16PtrFromString("open");if e!=nil{return e}
 p,e:=windows.UTF16PtrFromString(path);if e!=nil{return e}
 return windows.ShellExecute(0,verb,p,nil,nil,windows.SW_SHOWNORMAL)
}

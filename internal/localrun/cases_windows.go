//go:build windows

package localrun

import (
 "errors"
 "os"
 "golang.org/x/sys/windows"
)
func hideDirectory(path string){
 p,e:=windows.UTF16PtrFromString(path);if e!=nil{return}
 a,e:=windows.GetFileAttributes(p);if e==nil{_ = windows.SetFileAttributes(p,a|windows.FILE_ATTRIBUTE_HIDDEN)}
}
func replaceCases(from,to string)error{
 a,e:=windows.UTF16PtrFromString(from);if e!=nil{return e}
 b,e:=windows.UTF16PtrFromString(to);if e!=nil{return e}
 return windows.MoveFileEx(a,b,windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}
func lockCases(path string)(func(),error){
 if st,e:=os.Lstat(path);e==nil&&!st.Mode().IsRegular(){return nil,errors.New("测试数据锁不是普通文件")}
 f,e:=os.OpenFile(path,os.O_CREATE|os.O_RDWR,0600);if e!=nil{return nil,e}
 var ov windows.Overlapped
 h:=windows.Handle(f.Fd())
 if e=windows.LockFileEx(h,windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,0,1,0,&ov);e!=nil{f.Close();return nil,errors.New("测试数据正在写入，请重试")}
 return func(){_ = windows.UnlockFileEx(h,0,1,0,&ov);_ = f.Close()},nil
}

//go:build !windows

package localrun

import (
 "errors"
 "os"
 "golang.org/x/sys/unix"
)
func hideDirectory(string){}
func replaceCases(from,to string)error{return os.Rename(from,to)}
func lockCases(path string)(func(),error){
 f,e:=os.OpenFile(path,os.O_CREATE|os.O_RDWR|unix.O_NOFOLLOW,0600);if e!=nil{return nil,e}
 if e=unix.Flock(int(f.Fd()),unix.LOCK_EX|unix.LOCK_NB);e!=nil{f.Close();return nil,errors.New("测试数据正在写入，请重试")}
 return func(){_ = unix.Flock(int(f.Fd()),unix.LOCK_UN);_ = f.Close()},nil
}

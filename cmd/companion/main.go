// Command companion provides the non-interactive local core.
package main

import (
 "context"
 "encoding/json"
 "flag"
 "fmt"
 "io"
 "os"
 "os/signal"

 "github.com/Routilet/companion/internal/localrun"
)
func run(args []string,out,errOut io.Writer)int{
 usage:=func(){fmt.Fprintln(errOut,"usage: companion test [-compiler PATH] [-case N] FILE | companion source FILE")}
 if len(args)==0{usage();return 2}
 if args[0]=="source"{
  if len(args)!=2{usage();return 2}
  p,e:=localrun.PrepareSubmission(args[1],"",nil);if e!=nil{fmt.Fprintln(errOut,e);return 2}
  if _,e=io.WriteString(out,p.Text);e!=nil{fmt.Fprintln(errOut,e);return 2};return 0
 }
 if args[0]!="test"{usage();return 2}
 fs:=flag.NewFlagSet("test",flag.ContinueOnError);fs.SetOutput(errOut)
 compiler:=fs.String("compiler","","compiler executable")
 index:=fs.Int("case",0,"1-based case number; 0 = all")
 if e:=fs.Parse(args[1:]);e!=nil||fs.NArg()!=1||*index<0{usage();return 2}
 path:=fs.Arg(0);cs,e:=localrun.LoadCases(path);if e!=nil{fmt.Fprintln(errOut,e);return 2}
 if *index>len(cs.Cases)&&!(len(cs.Cases)==0&&*index==1){fmt.Fprintln(errOut,"invalid case number");return 2}
 ctx,cancel:=signal.NotifyContext(context.Background(),os.Interrupt);defer cancel()
 r,e:=localrun.RunBatch(ctx,path,*compiler,cs.Revision,*index-1);if e!=nil{fmt.Fprintln(errOut,e);return 2}
 enc:=json.NewEncoder(out);enc.SetIndent("","  ");if e=enc.Encode(r);e!=nil{fmt.Fprintln(errOut,e);return 2}
 if r.Stale||r.State!="completed"{return 1}
 for _,c:=range r.Cases{if c.State!="passed"&&c.State!="ran"{return 1}}
 return 0
}
func main(){os.Exit(run(os.Args[1:],os.Stdout,os.Stderr))}

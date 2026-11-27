package localrun

import (
 "errors"
 "unicode/utf8"
)

type Submission struct {
 Source Source `json:"source"`
 Text string `json:"text"`
 Digest string `json:"digest"`
 SourceDigest string `json:"sourceDigest"`
 ResourceDigest string `json:"resourceDigest,omitempty"`
 KitVersion string `json:"kitVersion,omitempty"`
 Components []string `json:"components,omitempty"`
 CaseRevision string `json:"caseRevision"`
 ToolchainDigest string `json:"toolchainDigest"`
 Verified bool `json:"verified"`
}
func PrepareSubmission(path,compiler string,last *BatchResult)(Submission,error){
 s,e:=Open(path);if e!=nil{return Submission{},e}
 b,e:=readSource(s);if e!=nil{return Submission{},e}
 if !utf8.Valid(b){return Submission{},errors.New("源码不是 UTF-8，不能无损复制文本")}
 prepared,e:=PrepareSource(s,b);if e!=nil{return Submission{},e}
 p:=Submission{Source:s,Text:string(prepared.Bytes),Digest:prepared.PreparedDigest,SourceDigest:prepared.SourceDigest,ResourceDigest:prepared.ResourceDigest,KitVersion:prepared.KitVersion,Components:prepared.Components}
 cs,e:=LoadCases(s.Path);if e==nil{p.CaseRevision=cs.Revision}else{p.CaseRevision="unavailable"}
 if cc,e:=compilerFor(compiler,isCPP(s.Path));e==nil{p.ToolchainDigest=toolchainDigest(cc,s.Path)}else{p.ToolchainDigest="unavailable"}
 if last!=nil&&!last.Stale&&last.State=="completed"&&last.Source.Path==s.Path&&last.SourceDigest==p.SourceDigest&&last.PreparedDigest==p.Digest&&last.ResourceDigest==p.ResourceDigest&&last.CaseRevision==p.CaseRevision&&last.ToolchainDigest==p.ToolchainDigest&&p.CaseRevision!="unavailable"&&p.ToolchainDigest!="unavailable"&&len(cs.Cases)>0&&len(cs.Cases)==len(last.Cases){
  p.Verified=true
  for i,c:=range last.Cases{if c.Index!=i||c.State!="passed"||cs.Cases[i].Expected==nil{p.Verified=false;break}}
 }
 return p,nil
}
func CheckSubmission(p Submission,compiler string)(Submission,error){
 now,e:=PrepareSubmission(p.Source.Path,compiler,nil);if e!=nil{return Submission{},e}
 if now.Digest!=p.Digest||now.SourceDigest!=p.SourceDigest||now.ResourceDigest!=p.ResourceDigest||now.CaseRevision!=p.CaseRevision||now.ToolchainDigest!=p.ToolchainDigest{return Submission{},errors.New("源码、资源、样例或编译配置已变化；请重新预览")}
 return now,nil
}

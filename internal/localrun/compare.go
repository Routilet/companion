package localrun

import (
 "regexp"
 "strconv"
 "strings"
 "path/filepath"
)
type Difference struct {
 Line int `json:"line"`
 Column int `json:"column"`
 Expected string `json:"expected"`
 Actual string `json:"actual"`
 ExpectedParts []DiffPart `json:"expectedParts,omitempty"`
 ActualParts []DiffPart `json:"actualParts,omitempty"`
 Coarse bool `json:"coarse,omitempty"`
}
func normalize(s string)string{
 lines:=strings.Split(strings.ReplaceAll(s,"\r\n","\n"),"\n")
 for i:=range lines{lines[i]=strings.TrimRight(lines[i]," \t")}
 return strings.TrimRight(strings.Join(lines,"\n"),"\n")
}
func visible(s []rune)string{
 if len(s)==0{return "〈末尾〉"}
 if len(s)>80{s=s[:80]}
 return strings.NewReplacer(" ","·","\t","→","\n","↵\n").Replace(string(s))
}
func Compare(expected,actual string)*Difference{
 e,a:=[]rune(normalize(expected)),[]rune(normalize(actual))
 i,line,col:=0,1,1
 for i<len(e)&&i<len(a)&&e[i]==a[i]{if e[i]=='\n'{line++;col=1}else{col++};i++}
 if i==len(e)&&i==len(a){return nil}
 ep,ap,coarse:=diffParts(e,a)
 return &Difference{Line:line,Column:col,Expected:visible(e[i:]),Actual:visible(a[i:]),ExpectedParts:ep,ActualParts:ap,Coarse:coarse}
}
type Diagnostic struct{
 Severity string `json:"severity"`
 Message string `json:"message"`
 Raw string `json:"raw"`
 File string `json:"file,omitempty"`
 Line int `json:"line,omitempty"`
 Column int `json:"column,omitempty"`
}
var diagPattern=regexp.MustCompile("^(.+?):([0-9]+):(?:([0-9]+):)?[ \\t]*(fatal error|error|warning|note):[ \\t]*(.*)$")
func diagnostics(raw,snapshot,source string)[]Diagnostic{
 out:=[]Diagnostic{}
 for _,line:=range strings.Split(strings.ReplaceAll(raw,"\r\n","\n"),"\n"){
  if line==""{continue}
  m:=diagPattern.FindStringSubmatch(line)
  if m==nil{
   if len(out)>0{out[len(out)-1].Raw+="\n"+line}else{out=append(out,Diagnostic{Severity:"info",Message:line,Raw:line})}
   continue
  }
  row,_:=strconv.Atoi(m[2]);col,_:=strconv.Atoi(m[3]);file:=m[1]
  if filepath.Base(file)==filepath.Base(snapshot){file=source}
  out=append(out,Diagnostic{Severity:m[4],Message:m[5],Raw:line,File:file,Line:row,Column:col})
 }
 return out
}

type DiffPart struct{
 Text string `json:"text"`
 Changed bool `json:"changed"`
}
// Bounded LCS over Unicode code points. Large inputs use prefix/suffix
// highlighting to avoid quadratic memory and excessive DOM nodes.
func diffParts(e,a []rune)([]DiffPart,[]DiffPart,bool){
 p,s:=0,0
 for p<len(e)&&p<len(a)&&e[p]==a[p]{p++}
 for s<len(e)-p&&s<len(a)-p&&e[len(e)-1-s]==a[len(a)-1-s]{s++}
 left,right:=e[p:len(e)-s],a[p:len(a)-s]
 ep,ap:=[]DiffPart{},[]DiffPart{}
 add:=func(parts *[]DiffPart,text string,changed bool){
  if text==""{return}
  n:=len(*parts);if n>0&&(*parts)[n-1].Changed==changed{(*parts)[n-1].Text+=text}else{*parts=append(*parts,DiffPart{text,changed})}
 }
 // Bound payload as well as LCS work. Raw complete outputs remain in Process.
 if len(e)+len(a)>16000{return nil,nil,true}
 add(&ep,string(e[:p]),false);add(&ap,string(a[:p]),false)
 coarse:=(len(left)+1)*(len(right)+1)>1000000
 if coarse{add(&ep,string(left),true);add(&ap,string(right),true)}else{
  w:=len(right)+1;dp:=make([]int,(len(left)+1)*w)
  for i:=len(left)-1;i>=0;i--{for j:=len(right)-1;j>=0;j--{
   if left[i]==right[j]{dp[i*w+j]=1+dp[(i+1)*w+j+1]}else if dp[(i+1)*w+j]>=dp[i*w+j+1]{dp[i*w+j]=dp[(i+1)*w+j]}else{dp[i*w+j]=dp[i*w+j+1]}
  }}
  i,j:=0,0
  for i<len(left)||j<len(right){
   if i<len(left)&&j<len(right)&&left[i]==right[j]{add(&ep,string(left[i]),false);add(&ap,string(right[j]),false);i++;j++}else if i<len(left)&&(j==len(right)||dp[(i+1)*w+j]>=dp[i*w+j+1]){add(&ep,string(left[i]),true);i++}else{add(&ap,string(right[j]),true);j++}
  }
 }
 if s>0{add(&ep,string(e[len(e)-s:]),false);add(&ap,string(a[len(a)-s:]),false)}
 return ep,ap,coarse
}

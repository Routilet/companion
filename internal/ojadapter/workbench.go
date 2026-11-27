package ojadapter
import("context";"errors";"strings";"time")
func(a *Accounts)hasMethod(method string)bool{if a.session==nil{return false};for _,m:=range a.session.Capabilities().Methods{if m==method{return true}};return false}
type RecentContest struct{ID string `json:"id"`;Title string `json:"title"`;Course string `json:"course"`;Start string `json:"start"`;End string `json:"end"`;Status string `json:"status"`}
func(a *Accounts)RecentContests(ctx context.Context)([]RecentContest,error){
 a.mu.Lock();defer a.mu.Unlock();ctx,done:=a.context(ctx);defer done()
 if a.closed||a.session==nil||a.state.ID==""{return nil,errors.New("请先登录 Accoding")}
 if !a.hasMethod("contests"){return nil,errors.New("请更新 Accoding Adapter；仍可手动输入链接")}
 var r struct{AccountID string `json:"accountId"`;Contests []RecentContest `json:"contests"`}
 if e:=a.session.Call(ctx,"contests",struct{}{},&r);e!=nil{return nil,a.submitError(e,false)}
 if r.AccountID!=a.state.ID||len(r.Contests)>10{return nil,ErrProtocol}
 seen:=map[string]bool{};for _,v:=range r.Contests{if !numberID.MatchString(v.ID)||seen[v.ID]||v.Title==""||!validText(v.Title,4096)||!validText(v.Course,4096)||!validText(v.Status,128){return nil,ErrProtocol};seen[v.ID]=true;for _,s:=range []string{v.Start,v.End}{if _,e:=time.Parse("2006-01-02 15:04:05",s);e!=nil{return nil,ErrProtocol}}}
 a.verified=true;return r.Contests,nil
}
var ErrBatchUnsupported=errors.New("当前 Adapter 不支持整场刷新")
func(a *Accounts)ContestJudges(ctx context.Context,targets []Target)([]JudgeStatus,error){
 a.mu.Lock();defer a.mu.Unlock();ctx,done:=a.context(ctx);defer done()
 if a.closed||a.session==nil||a.state.ID==""{return nil,errors.New("请先登录 Accoding")}
 if !a.hasMethod("contest_status"){return nil,ErrBatchUnsupported}
 if len(targets)==0||len(targets)>200{return nil,ErrProtocol}
 contest:=targets[0].ContestID;ids:=[]string{};want:=map[Target]bool{}
 for _,t:=range targets{if t.Kind!="contest"||t.ContestID!=contest||t.URL()==""||want[t]{return nil,ErrProtocol};want[t]=true;ids=append(ids,t.ProblemID)}
 var out []JudgeStatus
 if e:=a.session.Call(ctx,"contest_status",map[string]any{"contestId":contest,"problemIds":ids},&out);e!=nil{return nil,a.submitError(e,false)}
 if len(out)!=len(want){return nil,ErrProtocol}
 for _,s:=range out{if !want[s.Target]||!s.valid(s.Target,a.state.ID,""){return nil,ErrProtocol};delete(want,s.Target)}
 a.verified=true;return out,nil
}
// Document text remains inert Markdown. Strip raw line breaks from title and
// preserve source link and original body; never include credentials or account.
func(p Problem)Markdown()string{
 if p.Statement==""{return ""}
 title:=strings.Join(strings.Fields(p.Title)," ")
 return "# "+p.Label+" · "+title+"\n\n来源：<"+p.SourceURL+">\n\n"+strings.ReplaceAll(p.Statement,"\r\n","\n")
}

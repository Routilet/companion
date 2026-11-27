package ojadapter
import("errors";"strings";"testing")
func TestManagerFailureClassification(t *testing.T){
 for _,c:=range []struct{raw,want string}{
 {"另一 Routilet 实例正在操作，请稍后重试","正在安装"},
 {"应用已禁用，请先在 Routilet 中启用","已禁用"},
 {"open /private/user/activation.json: no such file or directory","未安装"},
 {"open C:\\private\\activation.json: The system cannot find the file specified.","未安装"},
 {"原生文件校验失败","SHA256"},
 {"读取应用清单 /private/user/manifest.json: denied","安装文件缺失"},
 {"secret token password","未识别错误"},
 }{e:=classifyManagerFailure(c.raw);if !strings.Contains(e.Error(),c.want)||strings.Contains(e.Error(),"private")||strings.Contains(e.Error(),"password"){t.Fatal(e)}}
 if !errors.Is(classifyManagerFailure("另一 Routilet 实例正在操作，请稍后重试"),errManagerBusy){t.Fatal("busy not distinguished")}
}

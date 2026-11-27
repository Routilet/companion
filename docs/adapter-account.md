# 阶段 02 · Companion 账号管理（开发预览）

设置增加 Accoding 一节。打开设置只读本地状态，不自动启动进程、读取系统凭据或联网。用户主动登录、使用已保存凭据、检查会话才启动/调用 Adapter；注销丢弃进程中的 Cookie 和身份，并删除已保存的凭据，不声称服务器端全局注销。失败、过期、离线或崩溃不会阻止本地工作，也不自动重登。账号错误显示稳定本地文本，不记录远程响应。关闭应用取消在途网络调用并回收进程；系统凭据授权对话框仍由操作系统控制。

开发预览仅接受受信任的绝对二进制路径 `COMPANION_ACCODING_ADAPTER_PREVIEW`；没有 PATH 搜索、自行下载或自动发现。设置显示 Adapter 不可用时，本地业务正常。正式发行前必须改为经 Routilet 校验且持有租约的激活快照路径；本阶段不改变安装器和分发。

默认不保存密码。明确勾选保存才将成功登录的用户名、密码存入 Windows Credential Manager（generic/local machine）或 macOS Keychain（系统 Security API，需要 CGO）；其他平台禁用。保存内容不写课程目录、配置文件、环境变量、命令行或日志。凭据库只有一个账号槽；显式新登录先清除旧槽，换号失败不会恢复旧账号。用户可以点击使用已保存凭据（即使启动时不检查是否存在）主动恢复。系统授权拒绝/保存失败如实提示，不回退明文。注销失败清理凭据时可重试；退出应用不删除选择保存的凭据。未对进程内存作不可验证的零化保证；不是防同一用户恶意程序的安全沙箱。

验收：Linux Go 1.23.2 隔离宿主包 race/vet、真实 Adapter 握手及模拟登录/换号/过期/离线/恢复/保存失败测试；Windows amd64 编译及 vet、macOS arm64 CGO=0 fallback 测试二进制交叉编译；独立 UI 模块 strict TypeScript 检查。完整 Wails/原生 Keychain 编译由双平台 CI；Windows CI 增加随机独立服务名的原生凭据读写/替换/删除测试；macOS 在有可用登录钥匙串时显式 `COMPANION_TEST_SYSTEM_CREDENTIALS=1 go test -run TestSystemCredentialRoundTrip ./internal/ojadapter`（不得使用实际账号）。真实 Accoding 联调、系统提示、实际安装发现、GUI 真机验收未完成。仍未实现题目导入、绑定、在线提交与评测。

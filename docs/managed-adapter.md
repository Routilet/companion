# 正式 Adapter 发现和运行保护

Companion 使用 Routilet 启动时注入的绝对 ROUTILET_EXECUTABLE，不搜索 PATH、不解析其他应用安装目录、不下载。通过 --adapter-call-probe 确认 routilet-adapter-call/1，再以 --adapter-available accoding-adapter 离线校验可用性；按需使用 --adapter-stdio accoding-adapter 连接，仍做 Adapter 自身 JSON Lines 协议/身份/能力握手。管理器注入路径存在时不回退开发预览路径；只有没有管理器路径时允许显式 COMPANION_ACCODING_ADAPTER_PREVIEW。

启动后异步检查一次、打开设置主动检查、登录/恢复前重新校验。普通状态/文件树缓存查询不启动管理器、不联网；不自动登录。现有运行会话由管理器保护，即使当前激活版本被移除也可结束已有会话。正常关闭先发送 EOF、关闭本地读取端，最多等待 3 秒回收管理器；开发直连等待 500ms。超时终止直接拥有的进程，保留管理器保护标记，不能保证异常退出后立即清理孙进程或标记；提交中断仍按不确定结果处理，不自动重试。

固定 Routilet PR #8 和已发行 Adapter beta.1 源码提交运行 Windows/macOS 专项 CI。真实本地进程测试使用临时配置与清单，核对离线握手、激活记录删除期间的会话、正常 EOF 后运行标记回收；Routilet 自身 CI 负责真实 npm 包安装/卸载。没有真实 OJ 凭据或提交。保留原有本地功能与版本，不发布。

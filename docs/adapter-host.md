# Companion Adapter 宿主（阶段 01）

新增 `internal/ojadapter`，与 `internal/localrun`、桌面绑定、CLI 业务隔离；仅提供按需 Start / Capabilities / Call / Close。应用启动不会探测、运行或联网，现有本地操作不改。没有可见 GUI 入口；阶段 02 再调用。

协议见 Adapter 阶段 01 的 `docs/stdio-v1.md`。固定协议 `routilet-oj-adapter/1`，启动需要可信绝对路径、显式预期 Adapter ID，不搜索 PATH、执行 shell 或自行安装；验证版本标记及 capabilities。不实现安装器解析或租约，正式使用前必须协调已有统一安装器并校验激活快照。工作目录为二进制所在目录；无文件路径或账号数据传递。stderr 不记录，以免泄漏敏感内容；错误返回仅稳定代码，不记录原始 JSON。

单进程串行，握手 5 秒、调用 60 秒（可缩短），最大帧 8 MiB；在途取消、超时、断线、错误 ID、协议异常均停止进程，关闭双向管道并等待退出。普通业务错误不关闭会话。排队取消不打断当前操作；显式 Close 幂等。无自动重启或重试。后续正式提交需在发送前记录未决尝试；取消不代表服务端撤销。本阶段不暴露 submit。强制结束宿主或不可信插件自行启动子进程不在回收保证范围内。

独立 CI 固定 Adapter 阶段 01 提交，分别在 macOS arm64 / Windows amd64 构建真实二进制，运行宿主测试及 vet，不发布。Linux 本地 Go 1.23.2 使用独立临时模块（只复制该包，不改 Companion go.mod），真实二进制联调、模拟故障、race、vet 和两平台测试二进制交叉编译通过；完整 Companion 原生构建依赖既有 CI，真机尚未验证。协议仅 capabilities，账号、导入、提交、评测状态、安装发现后续实施。

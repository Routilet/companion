# 独立仓库迁移

迁移基线：Routilet/accoding-companion @ 3161fc14d9c955e9e1ef977dfe5cbd72aa774aa4，来自阶段 01–05 的堆叠 PR #26–#35。原始仓库和分支保留完整历史；新仓库只复制 next/ 的源码和设计文档、交互原型、MIT LICENSE，不包含旧 TUI、OJ 登录/提交、旧发布脚本。Go module 路径改为 github.com/Routilet/companion，工程移至仓库根。CI 为纯构建/测试，固定真实 Exam Kit 和 Observer 发行夹具；已核对通过的 Windows/macOS CI 的 go.mod、go.sum、package-lock.json，三者分别字节一致（run 45966035444），提交锁文件；CI 强制 npm ci 和 go.mod/go.sum 无变更。

原仓库 PR 未合并、旧包不变。新仓库 main 仅初始化说明，迁移通过独立 PR 审核。阶段 06 的 release.yml、原生 manifest、自动资源发现和 Routilet catalog 启用另行实施；不提前发布。保留阶段 05 临时预览接口及本地测试。

macOS 及 Windows 真机验收尚未完成。阶段 05 修复 Observer capability backends 与 session backend 字段不同（gdb-mi/lldb-dap vs gdb/lldb），须等待真实双平台 CI 验证。旧版文档中的路径和阶段描述属于历史设计记录；以本仓库 README 与迁移说明为准。

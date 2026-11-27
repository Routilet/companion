# 原生发行和 Routilet 集成

本阶段候选版本 0.1.0-beta.1（尚未发行）。根 package.json 为 private，打包脚本在 staging 去除 private，正式包遵循 manifest v1（version、platforms executable/sha256），固定依赖 @routilet/c-exam-kit 0.6.10。无安装脚本、无 Node 运行要求。不依赖 Observer npm 包。启动后基于自身可执行文件真实路径和所属 package.json 解析固定 Exam Kit 依赖（嵌套或 pnpm virtual store / npm 同级），不扫描 cwd、不联网。已有快照照常使用；缺失提示修复。开发者仅可用 COMPANION_EXAM_KIT_PACKAGE 回退预览。

Observer 使用 Routilet PR #6 的 app-call/1：先 --app-call-probe，再 --query-app observer -- --capabilities，验证正式 v0.2.0、平台、backend、session v1/features；--open-app observer --wait -- --session FILE 由 Routilet 保护运行快照。没有 Routilet 或 Observer 不影响本地功能。失败等待可能不明确，保留会话并显示目录；成功退出才清理。Routilet 查询与启动之间仍可能更换激活版本，协议要求调用者验证兼容，不能保证原子版本绑定。正式 UI 移除两个手动包目录按钮。开发夹具的直接 ProbeObserver 保留供测试。

CI 验证真实发行资产和安装清单；实际管理器启动、中文路径、真机 GUI 仍需验收。暂不启用 catalog，直到兼容测试包可用；不发布。

## 图标及安装回归

组织设计源 `Routilet/.github/assets/icons/companion.svg` 的字节相同副本为唯一输入；前端标题栏使用原 SVG，Wails 运行时图标、ICO、ICNS 均从它生成。无需设计新图标。安装测试下载与 Routilet 一致的 pnpm 12.22.0 原生发行包并校验 SHA256，使用相同 install 参数及隔离配置，避免依赖 npm 包的内部 CLI 路径。macOS 安装后再次验证应用签名。真实窗口、任务栏及 Dock 外观仍需真机验证。

## 首次测试版授权

维护者明确批准合并并发布 0.1.0-beta.1 供真机验收。release.yml 增加仅该版本的 main push 自动发布条件；必须经过双平台构建与安装验证，推送 next，不变更 latest，已存在版本不重复发布或改标签。其他版本仍需 main 手动勾选 publish。此条件不是常规正式版自动发布策略。

## beta.2 交互回归

新建、重命名等改用应用内部 HTML dialog，移除所有浏览器原生 prompt/confirm。观察按钮直接启动，立即显示进度，保留真实后端错误。新增实际前端的 Chromium 自动交互回归（模拟 Wails RPC）；实际系统 WebView 和 Observer GUI 仍需双平台验收。beta.2 尚未发行。

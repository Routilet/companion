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

## beta.2 工作区优化

遵循 Observer 的单一主区、直接操作、就地状态、信息去重和稳定布局原则。固定顶栏，独立滚动的可调整文件树及顶部对齐工作区；选中文件的操作菜单、样例菜单收纳低频动作；编译器与版本移入设置。样例标签承担选择及状态，主区只显示当前选中样例的最新批次结果，后端仍只缓存最近一次批次（单次运行不累计历史批次）。修改后保守标记整次结果过期，冻结输入/预期可按需检查。失败默认逐行对比（按现有 Go 比较规则规范化，界面仅展示差异，不重新判定结果），保留原始文本；长输出限制表格预览范围。手动保存与运行前保存不变，不引入自动保存。视觉参数取自 Observer，沿用现有品牌图标和前端技术栈。真机验收继续进行。维护者已明确批准 beta.2 发布；main push 自动发行只限 0.1.0-beta.2，并要求前端与双平台安装测试通过，正式版仍需手动授权。

## beta.3 联合验收与测试发行

维护者已授权 0.1.0-beta.3 测试发行。沿用 Exam Kit 0.6.10，Observer 与 Accoding Adapter 独立可选；正式 Adapter 调用对齐 Routilet 0.1.6（固定源码 SHA 0405ed8）。发行工作流必须通过实际前端、双平台构建/候选安装及最终管理器/Adapter 原生联合测试，检查已发布的 Routilet 0.1.6 和 Adapter 0.1.0-beta.1 npm 版本，才发布 beta.3 到 next。已存在版本不重新发布；不修改 latest。不再自动发布旧 beta.2。真实平台与真机验收待进行。详见 [发行说明](release-beta.3.md)。

## beta.4 独立 Exam Kit（待审核）

移除固定 npm 依赖。启动时通过 Routilet 0.1.7 的 `--resource-probe` / `--resource-export c-exam-kit` 获取独立资源副本，校验协议、固定文件清单、大小及既有完整业务 schema，存放会话私有临时目录，正常退出清理。已打开会话不随更新或禁用改变；重新打开后使用当前激活版本。无资源时仍允许普通源码、已有快照功能，不自动下载、不隐式回退。仅无 manager 时允许显式 COMPANION_EXAM_KIT_PACKAGE 开发路径。旧依赖定位函数保留作历史兼容测试，不参与新启动路径。候选安装 CI 用独立测试资源验证，无生产依赖；还需真实 Routilet 联调。beta.4 不自动发布，待维护者审核。

## beta.4 发行授权

维护者授权测试发行 0.1.0-beta.4，包含独立资源和 Adapter 可用性修复。固定 Routilet 0.1.7，联合测试包含资源导出、会话副本和版本快照；资源缺失时新建普通 C 文件。仅发布 next；保留其他版本的显式授权流程。真实 OJ 登录和系统 WebView 仍待用户测试。

## beta.5 发行授权
维护者授权一次性测试发行 Companion 0.1.0-beta.5 和 Accoding Adapter 0.1.0-beta.2，保留 Routilet 0.1.7 与独立 Exam Kit/Observer。仅 next；原有文件及绑定不迁移。主线发行须通过实际前端、两平台构建、候选安装和新 Adapter 管理器联合测试，并核对 npm 上新 Adapter 可见后发布。参见 [工作台规范](workbench-beta5.md) 与 [发行说明](release-beta.5.md)。

## beta.6 发行授权
维护者授权 Companion 0.1.0-beta.6 测试发行，配套 Routilet 0.1.8 启动锁修复；Adapter 0.1.0-beta.2、Exam Kit 0.6.10、Observer 无需新版本。资源按需读取并在单次操作内固定；旧快照保持。主线发布门禁包括实际浏览器、双平台原生构建与安装、固定新版管理器和 Adapter 联调及 npm 依赖可见性。仅 next，不覆盖旧版本。详见 [beta.6](release-beta.6.md)。

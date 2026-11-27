# Companion Next · 05 Observer 协作

基于 #30。普通 C99/C++17 源文件直接打开，无需初始化；真实目录树、测试数据、批量运行、诊断、差异与结果失效。03d 新增原生 Wails 文件/目录拖放（一次一个）、恢复最近目录/相对文件、全局 C/C++ 编译器和外部编辑器设置、打开编辑器/所在目录、普通源码预览与剪贴板复制、非交互 CLI、批量进度和 Unicode 字符差异。设置存用户配置目录，不污染课程目录；不保存测试结果。旧数据不兼容、不读取。

## CLI

开发环境在 next/ 下运行 go build -o out/companion-cli ./cmd/companion（Windows 加 .exe）。分发名暂为 companion-cli，最终由 Routilet 确定。

- companion-cli test [-compiler PATH] [-case N] FILE：使用已保存的本地样例，默认全部；编号从 1 开始；无样例则空输入运行。stdout 为 JSON，stderr 为错误；0 = 所运行组成功/无预期正常退出，1 = 失败/取消/过期，2 = 参数或工具错误。不代表 OJ AC。
- companion-cli source FILE：输出原样 UTF-8 独立普通源码，不修改文件；不自动展开头文件，Exam Kit 留 04。

## 构建与验收

Go 1.26、Node 22、已有 GCC/Clang。desktop/frontend 执行 npm ci --ignore-scripts --no-audit --no-fund && npm run build；next/ 执行 go mod tidy、go test ./internal/localrun ./cmd/companion、go vet ./internal/localrun ./cmd/companion、go build -tags production ./desktop。独立双平台 CI 仅上传开发 ZIP，依赖锁随 CI 产物。未发布。

人工验收：拖入单文件/目录、取消未保存切换、重开恢复子目录选中、编译器设置跨重启；指定编辑器路径并打开、打开所在目录；复制原样源码、预览期间外部修改则拒绝复制、未通过全部比较仍允许明确复制；批量运行观察完成数及取消；Unicode 差异、多处差异及长输出降级。CLI stdout/退出码、无写入行为。真机 Windows/macOS 未验收。

## 实现限制

- 进度按 300ms 轮询；只显示编译与已完成用例数，仍一次返回完整批次。拖放由原生事件入队，忙时拒绝，需真机验证 WebView。外部编辑器仅支持单一可执行文件路径/名称，不解析 shell 或自定义参数；macOS 默认文本编辑器使用 open -t，Windows 使用文件关联。
- 字符差异以 Unicode code point 的 LCS 对齐，使用规范化比较文本；矩阵超过 1M 单元降级为前后缀/整段差异，超过 16000 字符只报告首差异，原始 stdout 仍保留。不是字素簇 diff。
- 复制快照重新核对源码、样例与编译配置；验证标记仅当最新完整批次所有样例均比较通过，不能证明独立源码对所有输入正确。普通 include 检查不是完整预处理器；系统剪贴板不保证粘贴字节恒等。运行可信源码，不是沙箱。
- 继承 03b 的 Windows Job Object 接管窗口/降级、03c 的轮询间隙及硬链接重命名限制；不支持多文件工程。03c/03d 仍需双平台真机检查；配置并发写采用最后一次保存，未实现配置合并。现有旧版不受影响。

## 04a：资源准备与默认新建 C

现行 @routilet/c-exam-kit 包版本 0.6.10（CI 固定资源仓库提交 f17f8715c6ca552b059da4f95969d956026e584a）使用 preset/manifest.json schema 2 内嵌组件和 reference.json schema 1；发布包不提供 expanded.h，Companion 按旧版 renderHeader 规则生成。只支持实际 v2，不兼容旧 .accoding 或旧 v1 包。保留许可证、引用数据以及可选 Markdown；无 Markdown 时不伪造。

预览安装包路径由 COMPANION_EXAM_KIT_PACKAGE（绝对路径）或设置内一次性目录选择提供；可从 04a CI 的 companion-04a-exam-kit-preview artifact 解压获取。不是正式 Routilet 接口，不执行 npm、网络访问或安装；正式激活资源定位留 06。新建 C 默认引入当前资源版本；资源缺失时不创建源码。C++ 普通模板不变。资源放在当前浏览根 .companion/resources/<version>/；子目录文件生成相对 include；同版本只复用一致的快照，新版本并存。完整头、manifest、reference、许可证和 snapshot.json 指纹记录一并保存；锁与暂存目录后发布，外部非协作写入仍可能竞争。指纹用于完整性检测，不是签名。旧代码不修改，删除安装包不影响本地快照。

验收：选择实际 npm 包目录，创建 C，外部 IDE 编译 read_int 示例；换版创建另一个 C，旧 include 不变；缺包、循环/未知依赖、同版本冲突、篡改头文件应失败。CI 使用真实生成的资源包和外部 GCC/Clang 编译新文件，另有合成 fixture 验证版本及损坏。04b 已接入含 Kit 源码的测试与复制；手册界面在 04c。GUI 真机仍待验收。Observer/Routilet/OJ 未接入，不改正式发行。stacked PR。

## 04b：统一装配

读取源码中的明确相对 include，验证祖先目录中的版本快照及内容。普通 C/C++ 仍保留原字节；Kit 源码采用受限 C99 单翻译单元规则：仅 #include 指令，不支持宏、条件编译、反斜杠续行、其他本地头或替代预处理符号。识别公开符号（跳过注释、字符和字符串），按依赖闭包及稳定拓扑选组件，检查保留函数冲突。注释用等宽空格替代以保留 token 与行号；生成的独立源码不含本地 include 或诊断 #line。不是通用 C 解析器，标识符保守匹配可能多选，复杂声明或符号重载式模式不保证精确识别；最终以正式编译为准。资源声明中的预处理指令按受信任 manifest 原样保留。

RunBatch/GUI 复制/CLI source 共用 PrepareSource；正式编译的 bytes 与提交快照一致。记录原始、装配、资源摘要及资源版本，变更或损坏使结果过期并阻止复制旧预览；不读取安装包来替换旧资源。映射编译器行号至用户源或组件逻辑位置，不再额外编译诊断版本；原始 stderr 保留，列位置依赖编译器，组件逻辑位置不是磁盘路径。空样例不宣称验证；没有完整通过仍允许明确复制。CI 增加合成与真实包的最小展开、实际执行、摘要一致、资源篡改、符号冲突、诊断映射回归。没有 OJ/手册 UI/分发；真机仍待验收。

## 04c：离线函数手册

顶栏函数手册 / Ctrl+H（macOS Cmd+H 可能被系统占用）在未打开源码时也可使用。读取当前 C 文件明确引用的版本快照；没有 Kit 引用或 C++ 则使用已安装资源。当前引用损坏时明确失败，不静默切换新版本；源码编译错误不妨碍查阅。直接读取 reference.json 原有 sections/categories、使用约定、相关操作、类型变体、函数签名、显式容量扩展接口和示例。搜索在 Go 核心进行，支持函数名、中文关键词、多词、精确变体优先；复制由本地剪贴板服务按条目和变体读取原始示例，复核资源指纹。使用对话框，保留当前文件及未保存样例；不访问网络，不自动写入源码。Markdown 采用有限的安全文本渲染（标题、行内代码、粗体、代码块），不执行 HTML、不打开外部链接；不是通用 Markdown 引擎。真机输入法、剪贴板、窄窗口及快捷键需验收。

CI 使用固定真实 Exam Kit 构建产物，验证目录结构、实际条目与类型搜索；合成测试验证版本并存、源码有误时查阅、快照删除安装包后可用、损坏不回退、非法目录及相关条目、示例选择。04c 仍未实现 Observer、Routilet 正式资源定位和 OJ。未合并或发布。

## 05：Observer 0.2.0（正式版）

权威协议：https://github.com/Routilet/c-observer/blob/v0.2.0/docs/integration.md 。不改 Observer 或 Routilet，不依赖 Node 运行。开发预览选择解压后的正式 npm 包目录（包含 manifest.json 和 bin/），或设置 COMPANION_OBSERVER_PACKAGE 为绝对路径；正式资源发现留 06。仅支持实际验证的 0.2.0、manifest/session v1、Windows amd64 GDB / Darwin arm64 LLDB。校验 manifest 相对路径、二进制 SHA256、--capabilities 的版本/平台/后端/feature。缺少 Observer 不影响测试。CI 下载固定正式 release tarball 并验证 release SHA256，检查实际二进制能力；不发布任何资源。

“观察当前样例”冻结磁盘当前 C 源码及所选已保存样例；未保存样例先保存，空列表使用空 stdin。不是历史测试快照。普通 C 复制原字节；Kit 使用同一次 PrepareSource 的组件选择和行映射，以 #line、sourceMap、skipSourcePaths 标识生成组件与冻结用户源码，恢复原始用户行以匹配 Observer 的源文本校验。Kit include 行含注释时保守拒绝观察。传 -lm，不传正式评测 -O2，Observer 使用自己的 -g -O0。workDir 为原目录，stdin/expected 分文件，null 省略，空字符串创建空文件；comparison=trim-line-end。无账号或 OJ。一次只启动一个独立观察窗口，进程退出后清理会话；Companion 关闭时等待 Observer 退出以保留冻结数据并避免强杀调试器，建议先关闭 Observer。启动/退出错误提示，程序本身运行结果由 Observer 展示。强制结束 Companion 可能遗留临时文件。工具链和 Windows ANSI 路径限制由 Observer 检查；非安全沙箱。

验收：用正式 0.2.0 包，普通 C 与 Kit C 分别观察有预期、无预期及空预期样例；检查真实 LLDB/GDB 源码映射、辅助组件跳过、-lm、取消、重跑与关闭，外部保存不改变冻结版本，返回后可继续测试。自动测试覆盖 manifest、能力探测、session 及冻结源码，不等价于 GUI 真机联调。Routilet 正式安装发现留 06；未合并未发布。

迁移后依赖锁以双平台 CI 的真实构建结果为准；不手工猜测校验值。
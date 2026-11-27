# Companion Next · 最小本地接口与运行语义（阶段 02）

状态：待审核；[存储规范](companion-next-storage.md)，[产品设计](companion-gui-product.md)。以下是语言无关服务契约，不是已实现的 Wails binding。Go 核心独立于 GUI；Wails 仅做对话框、剪贴板、事件适配。每次调用校验路径，不允许任意 shell 命令。无 OJ、插件、跨应用协议。

## 1. 对象与接口

文件路径是会话定位，绝非数据库 ID；相对路径以当前导航根为准，实际读写采用经验证的绝对路径。所有异步响应携带 sourcePath/runId，防止切换文件时渲染错位。

| 服务 | 操作 | 返回或行为 |
| --- | --- | --- |
| Files | OpenFile(path), OpenFolder(path), ListDirectory(path) | 根目录、当前文件、直接子项（名称/类型/可选状态）；单文件自动选父目录 |
| Files | CreateSource(directory,name), RenameSource(path,newName) | 新文件路径或明确冲突；新建 C 默认 kit（阶段 04） |
| Files | OpenInEditor(path), RevealDirectory(path) | 通过系统或显式编辑器打开，不接管编辑 |
| Cases | Load(path), Save(path,cases,revision) | 有序 Case[] 和存储修订号；冲突不覆盖 |
| Cases | ListOrphans(directory), Import(path,jsonPath,revision,append) | 显式复制恢复，不删除原数据 |
| Runner | StartTest(path,selection,caseRevision), Cancel(runId), GetResult(runId) | 当前/全部样例，返回 runId，进度/结果及取消 |
| Source | PrepareSubmission(path), CopySubmission(snapshotToken) | 预览与短期会话 token，已验证状态；复制前核验当前版本 |
| ExamKit | GetReference(path?), SearchReference(path?,query) | 当前文件资源版本或已安装版本、分类条目和签名/示例 |
| Settings | Get(), Save(settings), CheckToolchain() | 配置与实际可用性；不安装编译器 |

文件选择/拖放、剪贴板实现由桌面层提供，不进入核心。Observer 按后续已确认接口协作，暂不设计 StartObservation 传输协议。运行通知仅需要 fileChanged(path/kind)、testProgress(runId,phase,completed,total) 和 testFinished(runId)；不用事件总线。单窗口同一时刻只运行一个测试任务；忙时允许浏览，但运行所用样例快照不可变。

## 2. 数据契约

- Case = {input: string, expected: string | null}；顺序用索引。单次运行捕获不可变数组及其摘要；零组时可显式运行空输入（临时用例，不强制保存）。没有预期输出不是通过。
- PreparedSource = {sourcePath, language, bytes, sourceDigest, resourceDigest?, preparedDigest, diagnosticMap?}。普通单文件源码保留原始字节（UTF-8，复制文本前验证编码）；kit 源码使用确定性展开。只在内存/临时目录保留。默认不剥除普通源码注释；kit 展开沿用旧规则去注释并保持 token 边界。正式编译和复制同一份 prepared bytes，不包含仅诊断使用的 #line。原始文件不改。
- RunResult = {runId, sourcePath, snapshotDigest, toolchainDigest, caseRevision, compilation, cases[], state}。Compilation 含 stdout、stderr、退出状态、结构化 diagnostics；CaseResult 含原始 input/expected/actual、stderr、耗时、退出状态、比较差异、状态。结果按快照中的索引归属，禁止将结果套到改变后索引。用户界面显示旧结果时明确过期。
- Diagnostic = {severity,message,raw,file?,line?,column?,origin?}，保留无法解析的原始信息，warning 不代表失败。映射只作显示，不改变正式编译判定。
- Error = {code,message,detail?}，区分无权限、文件缺失、数据损坏、修订冲突、工具链缺失、资源缺失/冲突、暂不支持的源码和进程错误。不吞异常、不返回假通过。

## 3. 测试管线

单翻译单元优先：标准库 + 可选一个受支持的 Exam Kit 引用；C99 / C++17。普通源文件支持正常预处理指令；首版不承诺自定义头文件、多文件工程或任意预处理展开。对于非 kit 的本地 include 或无法确认可独立复制的依赖，明确提示不属于独立提交模式；不假称已展开。首个实现切片可拒绝此类输入，后续有真实需求再支持本地项目测试。kit 展开沿用旧版受限语法，拒绝条件 include、未知依赖、保留符号冲突等不能可靠处理的情况，不静默生成错误源码；宏/条件编译支持范围单独在阶段 04 验证。普通单文件不因 kit 限制拒绝常规宏。

运行开始时捕获源、资源、编译配置、样例快照。正式源码写系统临时目录，一次编译，逐组独立执行；进程 cwd 为原源文件父目录，stdin 原样写入后关闭，stdout 与 stderr 分离；不运行 shell。默认使用现有工具链：显式设置 > CC/CXX（若可用）> Windows gcc/g++、macOS cc/c++；不写系统 PATH，必要时仅子进程添加 Windows 编译器目录。默认保留旧版 30s 编译、2s 每组运行、4 MiB stdout+stderr 合计上限，取消/超时/超限均停止进程树并清理，平台机制需真机验证。不是安全沙箱，仅运行用户主动选中的代码。

状态：未运行、运行中、通过、输出不一致、已运行（无预期）、编译失败、运行失败、超时、已取消、输出超限、工具链缺失。过期是独立标记，不抹掉历史结果。退出非零优先于输出相等；编译失败不运行用例。无预期不计入通过分母，所有无预期显示运行组数。

## 4. 比较规则

第一版只有一个默认规则，沿用旧版 trim-line-end：CRLF -> LF，忽略每行末尾 ASCII 空格与 tab、末尾空行；保留行首、行中空格和内部空行，不做 token 等价或浮点容差。空预期仍参与比较；仅有被忽略的空白输出可通过。界面注明这是本地比较规则，不是正式 OJ 判定。原始输出不被改写；差异定位使用规范化内容，显示空格/tab/换行及首个不同位置，原始文本可查看/复制。字符差异复用旧算法思路，长输出有上限。比较模式选择暂不增加。

## 5. 结果与复制一致性

sourceDigest 对磁盘原始字节；resourceDigest 覆盖快照必需文件；preparedDigest 对正式源码 bytes；toolchainDigest 覆盖解析后的编译器路径、参数及配置（不承诺识别同路径外部二进制替换的所有情况）；caseRevision 为样例内容摘要。每组结果对应其实际输入/预期；源、资源、配置变化使相关结果过期，样例变化保守使该文件所有结果过期。文件事件只是提示，运行及复制前重新读取验证，切换/恢复时亦检查；运行过程中外部保存不改变当前运行快照，结束时标为过期。没有自动运行。

PrepareSubmission 生成当前版本、显示是否存在匹配且全部比较用例通过的测试（无比较用例不得宣称已验证）；如果有未跑组、无预期组或失败，显示未完整验证，仍允许用户显式复制。CopySubmission 前再次核验 token、源/资源/配置/样例；若变化则要求重新预览，不偷偷复制不同 bytes。普通 UTF-8 源码直接复制原文，系统剪贴板可能规范化换行，因此字节一致承诺针对生成/编译的源码快照，平台复制粘贴不承诺字节恒等。用户手工复制到考试平台后由平台决定评测。 
   
CLI 仅保留独立价值的非交互 test <file> 与 source <file>（标准输出为准备好的源码，不写用户文件），错误走 stderr/退出码，具体参数和稳定性在阶段 03 实现时定稿。不迁移 TUI。

## 6. 实现顺序

03a 打开真实普通 C → 编译运行 → 输出；03b 样例保存、批量、比较、错误/取消；03c 文件树、创建/重命名/外部变化与过期（C 创建正式资源依赖在 04 完成）；04 kit 快照、手册、展开与复制；05/06 外部协作。每个切片独立 PR/回归；本阶段无运行代码，不提前建立通用插件或 RPC。

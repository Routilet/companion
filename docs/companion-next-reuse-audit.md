# Companion Next · 旧版资产复用审计（阶段 02）

审计基线：`95541ea4231d1ea53c514bf89ce52ebccd53d2ac`。旧代码作为行为参考，Go 重写不逐行翻译。以下由实际源码及测试声明核对，未执行旧测试，未测量工作量。产品基线 PR #26。

## 可复用规则与回归

| 源码位置（旧版） | 现状 | 新版处理 |
| --- | --- | --- |
| [judge.ts:6–30,70–200](https://github.com/Routilet/accoding-companion/blob/95541ea4231d1ea53c514bf89ce52ebccd53d2ac/src/judge/judge.ts#L6-L200) | 输出三种规范化模式、一次编译多组运行；原始 cwd、临时源码；Windows Unicode 相对产物路径、子进程 PATH | 保留 trim-line-end 默认、流程与错误分类；不迁移 OJ case id/enabled；临时目录所有退出路径清理 |
| [process.ts:10–116](https://github.com/Routilet/accoding-companion/blob/95541ea4231d1ea53c514bf89ce52ebccd53d2ac/src/judge/process.ts#L10-L116) | spawn 不经 shell，stdin 关闭，stdout/stderr 合计限制，POSIX 进程组、Windows taskkill | Go 原生进程实现及平台清理；不要照搬可能的进程退出竞争；新增取消、后代进程、管道与清理测试 |
| [line-diff.ts](https://github.com/Routilet/accoding-companion/blob/95541ea4231d1ea53c514bf89ce52ebccd53d2ac/src/standalone/line-diff.ts#L1-L72) | 字符片段差异 | 迁移算法行为及 Unicode/长行测试，不迁移终端渲染 |
| [compiler-diagnostics.ts:49–139](https://github.com/Routilet/accoding-companion/blob/95541ea4231d1ea53c514bf89ce52ebccd53d2ac/src/standalone/compiler-diagnostics.ts#L49-L139) | GCC/Clang 风格诊断、note、warning、源码上下文 | 保留结构化位置与 raw，映射失败回退，警告不阻断 |
| [scaffold.ts:51–188,189–327](https://github.com/Routilet/accoding-companion/blob/95541ea4231d1ea53c514bf89ce52ebccd53d2ac/src/standalone/scaffold.ts#L51-L327) | manifest v1 外部组件/v2 inline，校验依赖及路径，完整头与确定性拓扑，屏蔽注释/字面量、选择符号、拒绝冲突、正式与诊断源码分离 | 迁移现行资源格式、验证和展开规则；旧 assembler 只允许 #include，不声称通用 C 预处理器。当前资源是否需同时支持 v1 由阶段 04 用真实包确认；不等同兼容 .accoding |
| [preset-manager.ts:189–258](https://github.com/Routilet/accoding-companion/blob/95541ea4231d1ea53c514bf89ce52ebccd53d2ac/src/standalone/preset-manager.ts#L189-L258) | 指纹、版本快照、暂存后发布、相对 include；旧路径绑定 root | 复用思想；新路径 .companion/resources/<version>，以源码父目录解析；额外校验 expanded.h 与 manifest 内容，禁止同版静默改写 |
| [workflow.ts:503–634](https://github.com/Routilet/accoding-companion/blob/95541ea4231d1ea53c514bf89ce52ebccd53d2ac/src/standalone/workflow.ts#L503-L634) | prepare/test 与账号/绑定、console 输出耦合；诊断二次编译需验证一致 | 独立 prepare/test；正式编译一次，必要时诊断另行验证；扩大快照指纹至资源/配置/样例；普通源码不去注释 |
| [manual.tsx:168–225](https://github.com/Routilet/accoding-companion/blob/95541ea4231d1ea53c514bf89ce52ebccd53d2ac/src/standalone/manual.tsx#L168-L225) | reference schema 1、分类树/变体、校验、关键词搜索 | 保留内容格式、校验和搜索；不迁移 Ink 布局、键盘状态或 markdown 终端渲染 |
| [observer.ts:34–117](https://github.com/Routilet/accoding-companion/blob/95541ea4231d1ea53c514bf89ce52ebccd53d2ac/src/standalone/observer.ts#L34-L117) | 观察会话准备/启动 | 参考生命周期，等 Observer 新接口；不迁移 TUI 退出恢复 |

## 不迁移

workspace/schema/store/contest/case-files、workspaces.ts、migrate.ts 的课程绑定、官方样例同步事务、旧工作区转换；新版只有每文件样例 JSON，无 .accoding 兼容。旧 OJ/auth/submission 模块、账号状态、登录表单、平台缓存、提交轮询不进入本地核心，旧仓库历史保留。TUI/terminal/旧安装器、npm 运行时发现逻辑、课程资源升级和撤销不迁移。Routilet 安装接口后续单独联调。CLI 仅 test/source 候选。

## 回归计划

- 03：从 [compiler.test.ts](https://github.com/Routilet/accoding-companion/blob/95541ea4231d1ea53c514bf89ce52ebccd53d2ac/tests/standalone/compiler.test.ts) 迁移缺编译器、中文空格路径、C/C++、语法错误、诊断 note/unknown/Windows 路径、Unicode 和超长行 diff。增加空输入、空 expected/null、CRLF、进程树超时/取消/超限、批量独立运行、失败后清理。
- 03：从 workflow.test.ts 的 standalone C/C++ judge 行为提取无 OJ 本地测试；新增源/样例/配置变化、运行中编辑、旧结果不可复用、普通源码复制一致性、写冲突、孤立 JSON 恢复、无写打开。
- 04：从 [scaffold.test.mjs](https://github.com/Routilet/accoding-companion/blob/95541ea4231d1ea53c514bf89ce52ebccd53d2ac/tests/standalone/scaffold.test.mjs) 迁移完整头、只选所需组件、传递依赖、循环/越界、非法 include、保留符号、v2 inline 与离线 reference、注释和字面量；新增同版本不一致、版本并存、完整头篡改、子目录相对 include、安装卸载不影响已有快照。compiler.test.ts 的 #line 映射及原始上下文测试归 04。
- 05：observer.test.ts 只参考会话输入与清理，接口待独立协调。
- 07：双平台原生实机，验证 Unicode 编译器、Windows 子进程 PATH、进程树清理、原子替换、隐藏属性、剪贴板、资源离线可用。CI 单测不能替代真机。

本轮只文档，不开发 Go，不运行旧测试，不重命名包/仓库。旧实现与新需求存在冲突处以上述新存储/接口提案为准，须维护者审核。

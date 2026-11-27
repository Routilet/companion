# Companion

![Companion](desktop/frontend/icons/companion.svg)

Routilet 的文件优先 C/C++ 编程辅助工具。普通源文件和文件夹无需初始化；外部编辑器写代码，Companion 管理测试样例、本地编译运行、版本化 Exam Kit 资源、函数手册及独立提交源码。可选 Observer 独立运行观察，可选 Accoding Adapter 提供账号、题目导入、确认提交与评测状态。无运行时 AI 或 Node.js 依赖。

## 测试版

当前候选版本 0.1.0-beta.6，面向 Windows 11 x64 和 Apple Silicon macOS 12+（系统 WebView）。通过 Routilet 0.1.8 或兼容新版的测试通道安装 `@routilet/companion`；在线功能另行安装并启用 `@routilet/accoding-adapter` 0.1.0-beta.2。Exam Kit 由 Routilet 独立安装、更新，Companion 在创建、导入、阅读手册或显式切换资源时按需获取已启用资源；已有工作区快照不自动改写。未安装时仍可使用普通文件的本地功能。Observer 独立可选。没有 Adapter 或网络仍可使用本地功能，首次进入 OJ 工作台时可恢复已保存登录，不自动下载。在线提交必须明确确认，结果不确定时不要直接重发。

测试版真实 Accoding 平台与实际 WebView 尚待用户验收，不宣称正式稳定版。不迁移旧课程数据，不更改旧 npm 包。请遵守课程考试规定；运行可信源码，本地运行不是沙箱。

- [发行与集成](docs/distribution.md) · [beta.6 发行说明](docs/release-beta.6.md) · [产品原则](docs/product.md)
- [正式 Adapter 接入](docs/managed-adapter.md) · [设计与路线图](docs/companion-next-roadmap.md)
- [存储设计](docs/companion-next-storage.md) · [核心接口](docs/companion-next-api.md)
- [迁移来源与验收](docs/migration.md) · [开发与测试](docs/development.md)

未经维护者审批不发布其他版本。

- [OJ 工作台与存储规范](docs/workbench-beta5.md)

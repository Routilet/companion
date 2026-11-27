# Companion

Routilet 的文件优先本地 C/C++ 编程辅助工具。普通源文件和文件夹无需初始化；外部编辑器写代码，Companion 管理测试样例、本地编译运行、版本化 Exam Kit 资源、函数手册及独立提交源码。可选 Observer 独立运行观察。无 OJ 插件，无运行时 AI 或 Node.js 依赖。

## 状态

从 [Accoding Companion 的阶段 05 分支](https://github.com/Routilet/accoding-companion/tree/3161fc14d9c955e9e1ef977dfe5cbd72aa774aa4) 迁移的新工程；尚未正式发行。目标 Windows 11 x64、Apple Silicon macOS 12+；系统 WebView。旧版仍在原仓库，不迁移旧课程数据，不更改旧 npm 包。

- 新包：`@routilet/companion`（占位版本不是可安装应用）
- 正式分发候选采用固定 Exam Kit 依赖和 Routilet app-call/1。详见 [发行与集成](docs/distribution.md)。尚未发布。
- [设计与路线图](docs/companion-next-roadmap.md)、[存储设计](docs/companion-next-storage.md)、[核心接口](docs/companion-next-api.md)；历史实现审计仅作参考。
- [迁移来源与验收](docs/migration.md) · [开发与测试](docs/development.md)

未经维护者审批不发布。遵守课程考试规定；运行可信源码，不是沙箱。

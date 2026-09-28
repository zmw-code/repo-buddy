package cmd

// Version 是当前版本号；MVP 完成前保持 0.0.x-dev（需求文档第十四节里程碑）。
const Version = "0.0.1-dev"

// Usage 是子命令总览，与需求文档 10.1 节契约保持一致。
const Usage = `repo-buddy —— 仓库管家

已实现:
  repobuddy scan [path] [--json]  工作区扫描：已暂存/已修改/未跟踪/已忽略
  repobuddy cjk-check             中文宽字符对齐自检（诊断，风险 #1）
  repobuddy version               版本信息
  repobuddy help                  本帮助

待实现（契约见需求文档 10.1 节）:
  repobuddy                       进 TUI（默认入口）
  repobuddy tui [path]            进 TUI
  repobuddy ignore                .gitignore 候选规则与写入
  repobuddy risk                  敏感文件 / 大文件扫描
  repobuddy commit                勾选暂存 + 提交 + 推送
  repobuddy meta                  labels / topics / description
  repobuddy status                本地 + 远端状态
  repobuddy serve-mcp             以 MCP server 运行（v0.2）
  repobuddy setup                 首次向导 / 右键菜单注册与卸载

scan 的 --json 结构见 docs/json-contract.md。`

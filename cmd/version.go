package cmd

// Version 是当前版本号；MVP 完成前保持 0.0.x-dev（需求文档第十四节里程碑）。
const Version = "0.0.1-dev"

// Usage 是子命令总览，与需求文档 10.1 节契约保持一致。
const Usage = `repo-buddy —— 仓库管家

用法:
  repobuddy             进 TUI（默认，待实现）
  repobuddy tui [path]  进 TUI
  repobuddy scan [path] 工作区扫描：未跟踪/已修改/已忽略
  repobuddy ignore      .gitignore 候选规则与写入
  repobuddy risk        敏感文件 / 大文件扫描
  repobuddy commit      勾选暂存 + 提交 + 推送
  repobuddy meta        labels / topics / description
  repobuddy status      本地 + 远端状态
  repobuddy serve-mcp   以 MCP server 运行（v0.2）
  repobuddy setup       首次向导 / 右键菜单注册与卸载
  repobuddy version     版本信息

诊断:
  repobuddy cjk-check   中文宽字符对齐自检（风险 #1，见 README）

接口契约详见需求文档第 10 节。`

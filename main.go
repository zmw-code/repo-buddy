// repo-buddy —— 本地 TUI 仓库管理工具（暂定名「仓库管家」）
// 需求文档：TUI仓库管理工具-需求分析.md（权威版本）
// 本文件只做「解析 → 分发」，子命令实现在 cmd/ 包里（契约见需求文档 10.1 节）。

package main

import (
	"os"

	"github.com/zmw-code/repo-buddy/cmd"
)

func main() {
	os.Exit(cmd.Dispatch(os.Args[1:], os.Stdout, os.Stderr))
}

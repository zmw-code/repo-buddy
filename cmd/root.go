// Package cmd 承载 CLI 子命令实现（契约见需求文档 10.1 节）。
//
// 约定：main.go 只做「解析 → 分发 → 退出码」，一个子命令一个文件，
// 每个子命令的入口签名统一为 func(args []string, stdout, stderr io.Writer) int。
package cmd

import (
	"fmt"
	"io"
)

// Dispatch 按 args 分发子命令，返回进程退出码。
func Dispatch(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stdout, "TUI 入口（待实现，见需求文档第九节界面草案）")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, Usage)
		return 0
	}

	switch args[0] {
	case "version", "--version", "-v":
		fmt.Fprintf(stdout, "repo-buddy %s\n", Version)
	case "help", "--help", "-h":
		fmt.Fprintln(stdout, Usage)
	case "scan":
		return Scan(args[1:], stdout, stderr)
	case "cjk-check":
		return CjkCheck(args[1:], stdout)
	case "ignore", "risk", "commit", "meta", "status", "serve-mcp", "setup", "tui":
		// 对应需求文档 10.1 节的接口契约，逐个实现
		fmt.Fprintf(stdout, "[%s] 尚未实现 —— 契约见需求文档 10.1 节\n", args[0])
	default:
		fmt.Fprintf(stderr, "未知命令: %s\n\n%s\n", args[0], Usage)
		return 2
	}
	return 0
}

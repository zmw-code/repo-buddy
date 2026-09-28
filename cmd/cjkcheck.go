package cmd

import (
	"io"

	"github.com/zmw-code/repo-buddy/internal/tui"
)

// CjkCheck 执行中文宽字符对齐自检（需求文档 风险 #1）。
func CjkCheck(_ []string, stdout io.Writer) int {
	return tui.RunCJKCheck(stdout)
}

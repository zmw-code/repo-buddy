// Package tui —— TUI 界面层（需求文档第九节草案）。
//
// 当前只落地风险 #1（中文宽字符对齐）的地基：按终端「显示宽度」计量的
// 补齐 / 截断工具与自检。Bubbletea 主界面属 MVP 后期里程碑，届时在本包内展开。
package tui

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// DisplayWidth 返回字符串在终端中占用的列数：CJK / 全角按 2 列，
// ASCII 按 1 列，组合符号（如 e + U+0301）按 0 列。
//
// 它就是「按显示宽度计」的 len()——TUI 里所有对齐都必须走这个函数，
// 直接用 len()（字节数）必然错位。
func DisplayWidth(s string) int { return runewidth.StringWidth(s) }

// PadRight 右补空格至显示宽度 w；已超宽则原样返回。
func PadRight(s string, w int) string {
	if d := DisplayWidth(s); d < w {
		return s + strings.Repeat(" ", w-d)
	}
	return s
}

// PadLeft 左补空格至显示宽度 w；已超宽则原样返回（用于数字列右对齐）。
func PadLeft(s string, w int) string {
	if d := DisplayWidth(s); d < w {
		return strings.Repeat(" ", w-d) + s
	}
	return s
}

// Truncate 按显示宽度截断，超出部分以 "…"（占 1 列）结尾。
func Truncate(s string, w int) string {
	if DisplayWidth(s) <= w {
		return s
	}
	var b strings.Builder
	used := 0
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if used+rw > w-1 {
			break
		}
		b.WriteRune(r)
		used += rw
	}
	return b.String() + "…"
}

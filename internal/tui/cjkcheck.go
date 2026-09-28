package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// RunCJKCheck 执行「中文宽字符对齐」自检（需求文档 风险 #1），返回退出码（0 = 通过）。
//
// 自动断言覆盖：宽度模型、盒线闭合、表格列对齐、按宽度截断。
// 无法自动断言的是「终端与字体是否真按 2 列渲染 CJK」——第 4 节给出目视材料。
func RunCJKCheck(w io.Writer) int {
	fail := 0
	total := 0
	add := func(n int, failed bool) {
		total++
		if failed {
			fail++
		}
	}
	check := func(label string, sample string, got, want int) {
		status := "PASS"
		bad := got != want
		if bad {
			status = "FAIL"
		}
		add(1, bad)
		// 列宽用 PadRight（按显示宽度）而非 fmt 的 %-12s（按字符个数）——
		// 后者遇到中文标签会把整行推右，正是本自检要拦的错位。
		fmt.Fprintf(w, "  %s期望 %2d 实测 %2d  %-4s 样本=%s\n", PadRight(label, 14), want, got, status, sample)
	}

	fmt.Fprintln(w, "repo-buddy 中文宽字符对齐自检（需求文档 风险 #1）")
	fmt.Fprintf(w, "go-runewidth EastAsianWidth=%v（东亚歧义类字符按 %s 计，可用 RUNEWIDTH_EASTASIAN 覆盖）\n\n",
		runewidth.EastAsianWidth, map[bool]string{true: "2 列", false: "1 列"}[runewidth.EastAsianWidth])

	// ── [1] 宽度模型 ────────────────────────────────────────────────
	fmt.Fprintln(w, "[1] 显示宽度模型（DisplayWidth；本报告的列宽也由它计算）")
	widthCases := []struct {
		label, sample string
		want          int
	}{
		{"ASCII", "abc", 3},
		{"CJK 汉字", "中文", 4},
		{"中英混排", "a中b", 4},
		{"全角标点", "，。！？", 8},
		{"半角标点", "a, b", 4},
		{"盒线字符", "┌─┐│", 4}, // ┌ ─ ┐ │ 四个字符，默认 EastAsianWidth=false 下各占 1 列
		{"组合音标", "e\u0301", 1},
	}
	for _, c := range widthCases {
		check(c.label, c.sample, DisplayWidth(c.sample), c.want)
	}
	// Emoji 各家终端不一致，只报告不断言
	fmt.Fprintf(w, "  %s报告 %2d（不断言：各终端/字体不一）  样本=%s\n", PadRight("Emoji", 14), DisplayWidth("🚀"), "🚀")
	fmt.Fprintf(w, "  %s报告 %2d（不断言：东亚歧义类，中文环境可能按 2 列渲染）  样本=%s\n\n",
		PadRight("歧义字符", 14), DisplayWidth("①±°→"), "①±°→")

	// ── [2] 盒线闭合 ────────────────────────────────────────────────
	fmt.Fprintln(w, "[2] 盒线闭合（Lipgloss 渲染后每行显示宽度必须相等）")
	box := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1).Render(
		strings.Join([]string{
			"文件名：中文需求文档.md",
			"状态：未跟踪（untracked）",
			"体积：340 MB / 1,204 个文件",
		}, "\n"))
	boxWidths := lineWidths(box)
	equal := allEqual(boxWidths)
	add(1, !equal)
	fmt.Fprintf(w, "  各行显示宽度: %v  → %s\n", boxWidths, passFail(equal))
	indented := indent(box, "  ")
	fmt.Fprintln(w, indented)

	// ── [3] 表格列对齐 ──────────────────────────────────────────────
	fmt.Fprintln(w, "[3] 表格列对齐（按显示宽度补空格）")
	const c1, c2, c3 = 22, 12, 10 // 三列显示宽度
	const sep = " │ "
	rows := []struct{ name, state, size string }{
		{"中文需求文档.md", "已修改", "12 KB"},
		{"node_modules/", "未跟踪", "340 MB"},
		{"src/main.go", "已暂存", "1.2 KB"},
	}
	wantLine := c1 + DisplayWidth(sep) + c2 + DisplayWidth(sep) + c3
	var table []string
	for _, r := range rows {
		line := PadRight(r.name, c1) + sep + PadRight(r.state, c2) + sep + PadLeft(r.size, c3)
		table = append(table, line)
	}
	tableWidths := lineWidths(strings.Join(table, "\n"))
	add(1, !allEqual(tableWidths))
	fmt.Fprintf(w, "  每行显示宽度: %v（合计应为 %d）  → %s\n",
		tableWidths, wantLine, passFail(allEqual(tableWidths) && tableWidths[0] == wantLine))
	// 列起点须三行一致
	starts := columnStarts(table, sep)
	add(1, len(starts) != 3)
	fmt.Fprintf(w, "  三列起点（显示宽度）: %v（须为固定值且三行一致）  → %s\n", starts, passFail(len(starts) == 3))
	fmt.Fprintln(w, indent(strings.Join(table, "\n"), "  "))
	// 同一份内容按字节数（len）补空格 → 必然错位，这就是必须用宽度库的原因
	fmt.Fprintf(w, "  对照：第一行按字节数计 %d、按显示宽度计 %d（len 会多算 %d 列，直接导致错位）\n\n",
		len(rows[0].name), DisplayWidth(rows[0].name), len(rows[0].name)-DisplayWidth(rows[0].name))

	// ── [3b] 反例 ───────────────────────────────────────────────────
	fmt.Fprintln(w, "[3b] 反例：按字节数（len）补空格的边框 —— 右侧竖线必然错位")
	bad := byteBox([]string{"中文需求文档.md", "node_modules/", "src/main.go"})
	fmt.Fprintln(w, indent(bad, "  "))

	// ── [3c] 截断 ───────────────────────────────────────────────────
	truncated := Truncate("中文需求文档-超级长的名字.md", 12)
	truncOK := DisplayWidth(truncated) <= 12 && strings.HasSuffix(truncated, "…")
	add(1, !truncOK)
	fmt.Fprintf(w, "[3c] 按宽度截断: %q → 显示宽度 %d  → %s\n\n", truncated, DisplayWidth(truncated), passFail(truncOK))

	// ── [4] 目视确认 ────────────────────────────────────────────────
	fmt.Fprintln(w, "[4] 目视确认（代码到此为止：终端与字体不在可控范围）")
	fmt.Fprintln(w, "  标尺 A（10 个汉字应占 20 列，其后 20 个点应刚好补齐）:")
	fmt.Fprintln(w, "    一二三四五六七八九十"+"·"+strings.Repeat("·", 19))
	fmt.Fprintln(w, "  标尺 B（下列矩形四条边框若不在直线上，说明该终端/字体宽度模型与本程序不同）:")
	fmt.Fprintln(w, indent(box, "  "))
	fmt.Fprintln(w, "  判定要点：")
	fmt.Fprintln(w, "    1. 上面两块方框的右边框必须上下成一条竖线；")
	fmt.Fprintln(w, "    2. 盒线字符（┌ ─ ┐ │）属 Unicode「东亚歧义」类：多数中文字体按 1 列渲染，")
	fmt.Fprintln(w, "       个别字体按 2 列。若踩中，TUI 需改用 ASCII 边框（+---+）或启动时探测；")
	fmt.Fprintln(w, "    3. 换字体 / 改字号 / 缩放窗口后请重跑本自检。")
	fmt.Fprintln(w)

	// ── 结论 ───────────────────────────────────────────────────────
	if fail == 0 {
		fmt.Fprintf(w, "结论：自动断言 %d 项全部通过 → PASS；第 4 节需人眼确认。\n", total)
		return 0
	}
	fmt.Fprintf(w, "结论：自动断言 %d 项，失败 %d 项 → FAIL（上方标 FAIL 的行即差异点）。\n", total, fail)
	return 1
}

// lineWidths 返回多行文本每行的显示宽度。
func lineWidths(s string) []int {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	out := make([]int, 0, len(lines))
	for _, l := range lines {
		out = append(out, DisplayWidth(l))
	}
	return out
}

func allEqual(xs []int) bool {
	for i := 1; i < len(xs); i++ {
		if xs[i] != xs[i-1] {
			return false
		}
	}
	return true
}

func passFail(ok bool) string {
	if ok {
		return "PASS"
	}
	return "FAIL"
}

// columnStarts 返回各列（以 sep 分隔）起点的显示宽度，并校验其余行与首行的列起点一致；
// 不一致时返回 [-1]，由调用方判 FAIL。
func columnStarts(lines []string, sep string) []int {
	if len(lines) == 0 {
		return nil
	}
	parts := strings.Split(lines[0], sep)
	starts := make([]int, 0, len(parts))
	pos := 0
	for _, p := range parts {
		starts = append(starts, pos)
		pos += DisplayWidth(p) + DisplayWidth(sep)
	}
	for _, line := range lines[1:] {
		if len(strings.Split(line, sep)) != len(starts) {
			return []int{-1}
		}
		pos = 0
		for i, p := range strings.Split(line, sep) {
			if i < len(starts) && starts[i] != pos {
				return []int{-1}
			}
			pos += DisplayWidth(p) + DisplayWidth(sep)
		}
	}
	return starts
}

// byteBox 用 len()（字节数）补空格画框，用于演示「按字节对齐」的错位。
func byteBox(lines []string) string {
	w := 0
	for _, l := range lines {
		if len(l) > w {
			w = len(l)
		}
	}
	var b strings.Builder
	b.WriteString("┌" + strings.Repeat("─", w+2) + "┐\n")
	for _, l := range lines {
		b.WriteString("│ " + l + strings.Repeat(" ", w-len(l)) + " │\n")
	}
	b.WriteString("└" + strings.Repeat("─", w+2) + "┘")
	return b.String()
}

func indent(s, prefix string) string {
	return prefix + strings.ReplaceAll(s, "\n", "\n"+prefix)
}

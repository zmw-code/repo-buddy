package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/zmw-code/repo-buddy/internal/core"
)

// scanListCap 是人读输出里每类最多列出的条目数；全量看 --json。
const scanListCap = 20

const scanUsage = `用法: repobuddy scan [path] [--json]

  path     仓库内任意目录（默认当前目录；会自动定位仓库根）
  --json   输出机器可读结构（结构见 docs/json-contract.md）

分类口径：已暂存（index 有改动）/ 已修改 / 未跟踪 / 已忽略。
只读操作：不修改任何文件，也不写 index。`

// Scan 实现 `repobuddy scan [path] [--json]`（契约见需求文档 10.1 节）。
func Scan(args []string, stdout, stderr io.Writer) int {
	var (
		path   string
		asJSON bool
	)
	for _, a := range args {
		switch {
		case a == "--json":
			asJSON = true
		case a == "-h" || a == "--help":
			fmt.Fprintln(stdout, scanUsage)
			return 0
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(stderr, "未知参数: %s\n\n%s\n", a, scanUsage)
			return 2
		default:
			if path != "" {
				fmt.Fprintf(stderr, "只能给一个路径（多余的: %s）\n\n%s\n", a, scanUsage)
				return 2
			}
			path = a
		}
	}
	if path == "" {
		path = "."
	}

	rep, err := core.Scan(context.Background(), path)
	if err != nil {
		if errors.Is(err, core.ErrNotRepo) {
			fmt.Fprintf(stderr, "不是 Git 仓库：%s\n", path)
			fmt.Fprintln(stderr, "提示：先把路径指向仓库内任意目录（含子目录），或在该目录执行 git init。")
			return 1
		}
		fmt.Fprintf(stderr, "扫描失败：%v\n", err)
		return 1
	}

	if asJSON {
		b, err := rep.JSON()
		if err != nil {
			fmt.Fprintf(stderr, "生成 JSON 失败：%v\n", err)
			return 1
		}
		stdout.Write(b)
		return 0
	}
	fmt.Fprint(stdout, renderScan(rep))
	return 0
}

// renderScan 把人读结果排版成文本。
func renderScan(rep *core.Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "仓库: %s  ⎇ %s\n", rep.RepoRoot, rep.Branch)
	fmt.Fprintf(&b, "已暂存 %d · 已修改 %d · 未跟踪 %d · 已忽略 %d\n",
		rep.Counts.Staged, rep.Counts.Modified, rep.Counts.Untracked, rep.Counts.Ignored)

	total := rep.Counts.Staged + rep.Counts.Modified + rep.Counts.Untracked + rep.Counts.Ignored
	if total == 0 {
		b.WriteString("\n工作区干净：没有变更，也没有被忽略的条目。\n")
		return b.String()
	}

	sections := []struct {
		name string
		list []core.Entry
	}{
		{"已暂存", rep.Staged},
		{"已修改", rep.Modified},
		{"未跟踪", rep.Untracked},
		{"已忽略", rep.Ignored},
	}
	for _, s := range sections {
		if len(s.list) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\n%s (%d)\n", s.name, len(s.list))
		for i, e := range s.list {
			if i == scanListCap {
				fmt.Fprintf(&b, "  … 另有 %d 条（用 --json 看全量）\n", len(s.list)-scanListCap)
				break
			}
			slash := ""
			if e.Dir {
				slash = "/"
			}
			fmt.Fprintf(&b, "  %s %s%s\n", e.Code, e.Path, slash)
		}
	}
	return b.String()
}

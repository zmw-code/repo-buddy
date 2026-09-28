// Package core —— 内核：扫描 / 忽略管理 / 提交推送 / 元数据 / 风险检测。
//
// 约定（需求文档第七节）：内核不依赖 UI，对外只暴露语言无关的接口 ——
// CLI 子命令 + JSON。本包所有操作默认**只读**：不动工作区文件、不写 index。
package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// SchemaVersion 是 --json 输出结构的版本号；v1.0 前该契约 unstable，
// 每次调整都要同步 docs/json-contract.md 与这里的版本号（需求文档 10.1 / 18.5）。
const SchemaVersion = 1

// ErrNotRepo 表示给定路径不在 Git 仓库内。
var ErrNotRepo = errors.New("不在 Git 仓库内")

// Kind 是工作区条目的分类。
type Kind string

const (
	KindStaged    Kind = "staged"    // index 有改动（git status 首列非空）
	KindModified  Kind = "modified"  // 仅工作区有改动
	KindUntracked Kind = "untracked" // 未跟踪
	KindIgnored   Kind = "ignored"   // 已被 .gitignore 忽略
)

// Entry 是工作区里的一个条目。
type Entry struct {
	Path string `json:"path"` // 相对仓库根，正斜杠
	Code string `json:"code"` // git 两位状态码原文，如 " M" / "??" / "!!"
	Kind Kind   `json:"kind"`
	Dir  bool   `json:"dir,omitempty"` // 仅 ignored：true 表示被折叠的目录
}

// Counts 是四类条目的数量。
type Counts struct {
	Staged    int `json:"staged"`
	Modified  int `json:"modified"`
	Untracked int `json:"untracked"`
	Ignored   int `json:"ignored"`
}

// Report 是 scan 的结果，也是 `--json` 的输出结构（见 docs/json-contract.md）。
type Report struct {
	Schema    int     `json:"schema"`
	Path      string  `json:"path"`      // 调用方给的路径（绝对、系统原生分隔符）
	RepoRoot  string  `json:"repo_root"` // 仓库根（绝对、系统原生分隔符）
	Branch    string  `json:"branch"`
	Counts    Counts  `json:"counts"`
	Staged    []Entry `json:"staged"`
	Modified  []Entry `json:"modified"`
	Untracked []Entry `json:"untracked"`
	Ignored   []Entry `json:"ignored"`
}

// Scan 扫描 path 所在仓库的工作区，按四类分组返回。
//
// path 可以是仓库内任意子目录（自动定位仓库根）；空串等价于当前目录。
// 只调用 `git rev-parse` 与 `git status`，不写任何文件。
func Scan(ctx context.Context, path string) (*Report, error) {
	if strings.TrimSpace(path) == "" {
		path = "."
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	out, err := git(ctx, abs, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrNotRepo, filepath.Clean(abs))
	}
	root := filepath.Clean(filepath.FromSlash(strings.TrimSpace(string(out))))

	// -z：路径以 NUL 分隔、不做引号转义 —— 中文/空格/特殊字符文件名都能原样拿到。
	// --ignored=matching：整目录被忽略时折叠成一条，避免 node_modules 里几万个文件刷屏。
	statusOut, err := git(ctx, root, "status", "--porcelain", "-z",
		"--untracked-files=all", "--ignored=matching")
	if err != nil {
		return nil, err
	}
	staged, modified, untracked, ignored, err := parseStatus(statusOut)
	if err != nil {
		return nil, err
	}

	rep := &Report{
		Schema:    SchemaVersion,
		Path:      filepath.Clean(abs),
		RepoRoot:  root,
		Branch:    branch(ctx, root),
		Staged:    staged,
		Modified:  modified,
		Untracked: untracked,
		Ignored:   ignored,
	}
	rep.Counts = Counts{
		Staged:    len(rep.Staged),
		Modified:  len(rep.Modified),
		Untracked: len(rep.Untracked),
		Ignored:   len(rep.Ignored),
	}
	return rep, nil
}

// JSON 返回 `--json` 的输出（缩进 2 空格，末尾带换行）。
//
// 关掉 HTML 转义：默认会把 `&` `<` `>` 写成 `\u0026` 之类，
// 于是路径 `docs/pro&sol.md` 在 `--json` 里变得不可读（解码结果其实一样，
// 但契约文档与人眼核对都需要原样）。见 docs/json-contract.md。
func (r *Report) JSON() ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// branch 返回当前分支名；游离 HEAD 时返回 "detached@<短 sha>"。
// symbolic-ref 在「仓库还没有第一个提交」时也可用（返回默认分支名）。
func branch(ctx context.Context, root string) string {
	if out, err := git(ctx, root, "symbolic-ref", "--short", "HEAD"); err == nil {
		return strings.TrimSpace(string(out))
	}
	if out, err := git(ctx, root, "rev-parse", "--short", "HEAD"); err == nil {
		return "detached@" + strings.TrimSpace(string(out))
	}
	return "（无分支）"
}

// parseStatus 解析 `git status --porcelain -z`（含 --ignored=matching）的输出。
//
// -z 记录格式：`XY <path>\0`，重命名/复制记录后紧跟一条「原路径」记录。
// 分类口径见 docs/json-contract.md：staged 与 modified 互斥（首列非空 → staged）。
func parseStatus(z []byte) (staged, modified, untracked, ignored []Entry, err error) {
	staged, modified, untracked, ignored = []Entry{}, []Entry{}, []Entry{}, []Entry{}
	records := bytes.Split(z, []byte{0})
	for i := 0; i < len(records); i++ {
		rec := records[i]
		if len(rec) == 0 {
			continue
		}
		if len(rec) < 4 || rec[2] != ' ' {
			return nil, nil, nil, nil, fmt.Errorf("无法解析 git status 记录: %q", rec)
		}
		code, p := string(rec[:2]), string(rec[3:])
		switch code {
		case "??":
			untracked = append(untracked, Entry{Path: p, Code: code, Kind: KindUntracked})
		case "!!":
			dir := strings.HasSuffix(p, "/")
			ignored = append(ignored, Entry{
				Path: strings.TrimSuffix(p, "/"), Code: code, Kind: KindIgnored, Dir: dir,
			})
		default:
			e := Entry{Path: p, Code: code}
			if rec[0] != ' ' {
				e.Kind = KindStaged
				staged = append(staged, e)
			} else {
				e.Kind = KindModified
				modified = append(modified, e)
			}
			if rec[0] == 'R' || rec[0] == 'C' {
				i++ // 重命名/复制：跳过紧随其后的「原路径」记录
			}
		}
	}
	sortEntries(staged, modified, untracked, ignored)
	return staged, modified, untracked, ignored, nil
}

func sortEntries(lists ...[]Entry) {
	for _, l := range lists {
		sort.Slice(l, func(i, j int) bool { return l[i].Path < l[j].Path })
	}
}

// git 在 dir 下执行 git 子命令，返回 stdout；失败时带上 stderr 的说明。
func git(ctx context.Context, dir string, args ...string) ([]byte, error) {
	full := append([]string{"-C", dir, "--no-optional-locks"}, args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return stdout.Bytes(), nil
}

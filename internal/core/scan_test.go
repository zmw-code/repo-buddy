package core

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseStatus 用固定的 porcelain -z 记录验证分类口径（不依赖 git 与文件系统）。
func TestParseStatus(t *testing.T) {
	z := strings.Join([]string{
		" M src/a.go",      // 仅工作区改动 → modified
		"?? 新文件.md",        // 未跟踪
		"!! node_modules/", // 被折叠的忽略目录
		"!! repo-buddy.exe",
		"MM src/b.go", // index + 工作区都有改动 → staged（信息在 code 里）
		"A  src/c.go",
		"R  src/new.go", // 重命名：紧随其后是原路径，必须被消费掉
		"src/old.go",
		"",
	}, "\x00")

	staged, modified, untracked, ignored, err := parseStatus([]byte(z))
	if err != nil {
		t.Fatalf("parseStatus 返回错误: %v", err)
	}

	assertEntries(t, "staged", staged, []string{"MM src/b.go", "A  src/c.go", "R  src/new.go"})
	assertEntries(t, "modified", modified, []string{" M src/a.go"})
	assertEntries(t, "untracked", untracked, []string{"?? 新文件.md"})
	assertEntries(t, "ignored", ignored, []string{"!! node_modules/", "!! repo-buddy.exe"})

	if len(ignored) > 0 && !ignored[0].Dir {
		t.Errorf("node_modules/ 应被标记为目录（dir=true）")
	}
	if len(ignored) > 1 && ignored[1].Dir {
		t.Errorf("repo-buddy.exe 不应被标记为目录")
	}
}

// TestParseStatusRejectsGarbage 确认解析器对不合格式的记录报错而不是静默吞掉。
func TestParseStatusRejectsGarbage(t *testing.T) {
	if _, _, _, _, err := parseStatus([]byte("垃圾记录\x00")); err == nil {
		t.Fatal("期望解析错误，实际为 nil")
	}
}

// TestReportJSON 验证 --json 契约的两条硬要求：可以原样看到 & < >（不做 HTML 转义），
// 且 counts 与四个数组的长度一致（见 docs/json-contract.md）。
func TestReportJSON(t *testing.T) {
	rep := &Report{
		Schema:    SchemaVersion,
		Path:      `E:\repo-buddy`,
		RepoRoot:  `E:\repo-buddy`,
		Branch:    "main",
		Untracked: []Entry{{Path: "docs/pro&sol.md", Code: "??", Kind: KindUntracked}},
	}
	rep.Counts = Counts{Untracked: len(rep.Untracked)}

	raw, err := rep.JSON()
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}
	if !strings.Contains(string(raw), `"docs/pro&sol.md"`) {
		t.Errorf("路径未原样输出（疑似被 HTML 转义）：%s", raw)
	}
	var back Report
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("输出不是合法 JSON: %v\n%s", err, raw)
	}
	if back.Counts.Untracked != len(back.Untracked) || back.Counts.Modified != len(back.Modified) ||
		back.Counts.Staged != len(back.Staged) || back.Counts.Ignored != len(back.Ignored) {
		t.Errorf("counts 与数组长度不一致：%+v", back.Counts)
	}
}

// TestScanWorkspace 在临时仓库里做端到端验证（需要 git CLI）。
func TestScanWorkspace(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	isolateGitConfig(t, dir)

	runGit(t, dir, "init", "-q", "-b", "main")
	writeFile(t, dir, ".gitignore", "node_modules/\n*.log\n")
	writeFile(t, dir, "kept.txt", "v1\n")
	runGit(t, dir, "add", ".gitignore", "kept.txt")
	runGit(t, dir, "commit", "-q", "-m", "init")

	writeFile(t, dir, "kept.txt", "v2\n")                 // 已修改
	writeFile(t, dir, "新增文件.md", "x\n")                   // 未跟踪
	writeFile(t, dir, "node_modules/pkg/index.js", "x\n") // 已忽略（整目录）
	writeFile(t, dir, "debug.log", "x\n")                 // 已忽略（文件）

	rep, err := Scan(context.Background(), dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if rep.Schema != SchemaVersion {
		t.Errorf("schema = %d，期望 %d", rep.Schema, SchemaVersion)
	}
	if rep.Branch != "main" {
		t.Errorf("branch = %q，期望 main", rep.Branch)
	}
	if !strings.EqualFold(rep.RepoRoot, filepath.Clean(dir)) {
		t.Errorf("repo_root = %q，期望 %q", rep.RepoRoot, filepath.Clean(dir))
	}
	if rep.Counts.Modified != 1 {
		t.Errorf("modified = %d，期望 1（%v）", rep.Counts.Modified, entryPaths(rep.Modified))
	}
	if rep.Counts.Untracked != 1 {
		t.Errorf("untracked = %d，期望 1（%v）", rep.Counts.Untracked, entryPaths(rep.Untracked))
	}
	if rep.Counts.Staged != 0 {
		t.Errorf("staged = %d，期望 0（%v）", rep.Counts.Staged, entryPaths(rep.Staged))
	}
	// ignored：debug.log 必在；node_modules 可能被折叠成一条目录，也可能逐文件列出
	if !hasIgnored(rep.Ignored, "debug.log") {
		t.Errorf("ignored 缺少 debug.log：%v", entryPaths(rep.Ignored))
	}
	if !hasIgnoredPrefix(rep.Ignored, "node_modules") {
		t.Errorf("ignored 缺少 node_modules：%v", entryPaths(rep.Ignored))
	}

	// 从子目录扫描：仍应定位到同一个仓库根
	sub := filepath.Join(dir, "node_modules", "pkg")
	rep2, err := Scan(context.Background(), sub)
	if err != nil {
		t.Fatalf("Scan(子目录): %v", err)
	}
	if !strings.EqualFold(rep2.RepoRoot, rep.RepoRoot) {
		t.Errorf("从子目录扫描得到 repo_root = %q，期望 %q", rep2.RepoRoot, rep.RepoRoot)
	}
}

// TestScanNotARepo 确认非仓库路径返回 ErrNotRepo。
func TestScanNotARepo(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	isolateGitConfig(t, dir)
	if _, err := git(context.Background(), dir, "rev-parse", "--show-toplevel"); err == nil {
		t.Skip("临时目录位于某个 Git 仓库内，跳过")
	}
	if _, err := Scan(context.Background(), dir); !errors.Is(err, ErrNotRepo) {
		t.Fatalf("err = %v，期望 ErrNotRepo", err)
	}
}

// ── 测试辅助 ────────────────────────────────────────────────────────

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("未安装 git，跳过")
	}
}

// isolateGitConfig 让测试不受本机全局/系统 git 配置影响（例如 autocrlf、全局 ignore）。
func isolateGitConfig(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(dir, "gitconfig-global"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	full := append([]string{
		"-C", dir,
		"-c", "user.name=repo-buddy test",
		"-c", "user.email=test@example.com",
	}, args...)
	if out, err := exec.Command("git", full...).CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertEntries(t *testing.T, label string, got []Entry, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %v，期望 %v", label, entryStrings(got), want)
	}
	for i := range want {
		if entryStrings(got)[i] != want[i] {
			t.Fatalf("%s = %v，期望 %v", label, entryStrings(got), want)
		}
	}
}

func entryStrings(es []Entry) []string {
	out := make([]string, 0, len(es))
	for _, e := range es {
		slash := ""
		if e.Dir {
			slash = "/"
		}
		out = append(out, e.Code+" "+e.Path+slash)
	}
	return out
}

func entryPaths(es []Entry) []string {
	out := make([]string, 0, len(es))
	for _, e := range es {
		out = append(out, e.Path)
	}
	return out
}

func hasIgnored(es []Entry, path string) bool {
	for _, e := range es {
		if strings.EqualFold(e.Path, path) {
			return true
		}
	}
	return false
}

func hasIgnoredPrefix(es []Entry, prefix string) bool {
	for _, e := range es {
		if strings.HasPrefix(strings.ToLower(e.Path), strings.ToLower(prefix)) {
			return true
		}
	}
	return false
}

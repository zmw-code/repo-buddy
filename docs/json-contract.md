# JSON 契约（`--json` 输出）

> 状态：**unstable** —— v1.0 前允许随迭代调整；每次调整必须同步本文档与 `schema` 版本号
> （需求文档 10.1 节 / 18.5 节）。
>
> 当前版本：`"schema": 1`
>
> 共同约定：
> - `--json` 的输出只走 stdout（人读内容不会混进来），失败信息走 stderr；
> - 字段名 snake_case，UTF-8 无 BOM；
> - 空列表输出 `[]`，绝不输出 `null`；
> - **不做 HTML 转义**：`&` `<` `>` 原样输出（默认的 `\u0026` 会让 `docs/pro&sol.md`
>   这类路径不可读；解码结果相同，但契约文档与人眼核对都要原样）；
> - 所有命令**只读**：不修改工作区文件、不写 index（设计原则 3）。

## 1. `repobuddy scan [path] --json`

扫描工作区，按四类分组返回。`path` 省略时为当前目录，可以是仓库内任意子目录（会自动定位仓库根）。

```json
{
  "schema": 1,
  "path": "E:\\repo-buddy\\docs",
  "repo_root": "E:\\repo-buddy",
  "branch": "main",
  "counts": {
    "staged": 0,
    "modified": 0,
    "untracked": 2,
    "ignored": 1
  },
  "staged": [],
  "modified": [],
  "untracked": [
    { "path": "docs/json-contract.md", "code": "??", "kind": "untracked" },
    { "path": "docs/pro&sol.md", "code": "??", "kind": "untracked" }
  ],
  "ignored": [
    { "path": "repo-buddy.exe", "code": "!!", "kind": "ignored" }
  ]
}
```

### 字段

| 字段 | 类型 | 说明 |
|---|---|---|
| `schema` | int | 结构版本号，当前 `1` |
| `path` | string | 调用方给的路径（绝对路径，系统原生分隔符） |
| `repo_root` | string | 仓库根（绝对路径，系统原生分隔符） |
| `branch` | string | 当前分支；游离 HEAD 时为 `detached@<短 sha>` |
| `counts` | object | 四类条目数，与下方四个数组长度一致 |
| `staged` / `modified` / `untracked` / `ignored` | array | 条目列表，按 `path` 升序 |
| `path`（条目内） | string | **相对仓库根**的路径，一律正斜杠（与 `.gitignore` 规则写法一致） |
| `code` | string | git 的两位状态码原文，如 `" M"` / `"??"` / `"!!"` |
| `kind` | string | `staged` / `modified` / `untracked` / `ignored` |
| `dir` | bool | 仅 `ignored` 可能出现：`true` 表示这是一条被折叠的目录 |

### 分类口径（与 `git status --porcelain` 的对应）

调用的是 `git status --porcelain -z --untracked-files=all --ignored=matching`：

| 输出码 | 归入 | 说明 |
|---|---|---|
| `??` | `untracked` | 未跟踪；`-uall` 保证逐文件列出，不会折叠成目录 |
| `!!` | `ignored` | 已忽略；整目录被忽略时 git 折叠为一条 `path/`（此时 `dir: true`） |
| 首列非空（如 `M ` / `A ` / `R ` / `MM`） | `staged` | index 内有改动；`MM` 表示同时还有未暂存的改动（信息在 `code` 里不丢） |
| 首列为空（如 ` M` / ` D`） | `modified` | 仅工作区有改动 |

- `staged` 与 `modified` **互斥**，一个条目只进一个数组；
- 重命名记录（`R`）之后的「原路径」记录已被解析器消费，不出现在任何数组里；
- `ignored` 的条目数 **不等于** 被忽略的文件数（git 会折叠整目录）。"忽略 N 个文件 / X MB"
  这类增量统计属于 S1（`ignore --suggest`）的职责，不在 `scan` 内做。

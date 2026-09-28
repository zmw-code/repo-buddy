# TUI 仓库管理工具 —— 需求分析

> 版本：v1.1（2026-09-27）
> 状态：已拍板（第十六节）；评审修订对照见第十八节
> 形态已定：**TUI only，明确不做 GUI**
> 本文档为权威版本，可直接作为 coding agent 的输入。

---

## 一、一句话定位

**一个跑在本地的 TUI 工具，把「本地文件现状」和「GitHub 仓库设置」放进同一个界面，用按钮而不是命令完成日常仓库管理。一次配置，此后右键/简单命令行/.exe即用。**

差异化三件事：
1. **本地 ↔ 远端打通**（相邻工具都只做一半）
2. **Windows 资源管理器原生入口**（免/少命令行操作）
3. **AI 时代专属的噪音/风险拦截**（如api_key/ai配置文件）

---

## 二、明确排除（先划红线，防止功能蔓延）

| 排除项 | 理由 |
|---|---|
| **任何形式的 GUI**（Qt / ImGui / Electron / Web） | **体量远大于 TUI**，与轻量工具定位冲突；Qt 动态分发要拖 ~100MB 运行时 |
| 完整 Git 客户端能力 | 交互式 rebase、bisect、worktree 交给 lazygit，不重复造轮子 |
| 代码编辑器 | 不做 IDE |
| 任何后端 / 云服务 | **零后端是红线**。一旦有服务器就同时背上成本与信任（Codeflow/Gitpod 的老路） |
| 文件同步 / 备份 | 超出范围 |
| 跨平台（首版） | **Windows-only 起步**，架构上不主动反跨平台，但不为它妥协设计 |

---

## 三、背景与痛点

AI 编程普及后，仓库维护的"手工成本"反而升高了：

1. **agent 一口气生成几十个文件**，工作区混着 `node_modules/`、`.venv/`、`__pycache__/`、`.claude/`、`.cursor/`、`.specstory/`、`dist/`、`*.log`——新手分不清哪些该提交、哪些该忽略
2. **agent 容易把密钥写进代码**（`.env`、`*apikey*`、`*.pem`、`id_rsa`），提交前没人拦
3. **GitHub 侧元数据**（labels、topics、description）只能去网页一个个点，API 能做但没图形入口，agent操作重点失真
4. **提交流程要敲一串命令**：`git add` → `git commit` → `git push` → `gh pr create`，新手记不住，老手嫌烦
5. **`gh`、`git` 都装好了**（本机实测：gh 2.99.0 已登录，git 凭据直接复用 gh），但缺一个把它们串起来的直观界面

> 市场结论（详见知识库《GitHub快捷图形化操作_市场与缺口》）：lazygit/gitui/TortoiseGit 只管本地，gh-dash/ghui 只管远端，gibo 只给模板不分析实际文件。**"看本地真实文件" + "改远端设置"的交集为空。**

---

## 四、目标用户与非目标用户

**目标用户**（给github的开发者）
- **A. AI 编程时代的普通用户**：用 agent 写代码，被一堆生成的文件搞糊涂，不知道怎么干净地提交
- **B. 熟练开发者**：知道该做什么，想要按键就能完成的操作，不想敲一串命令

**不是目标用户**
- 想要精细图形化拖拽界面的人（本项目没有 GUI）
- 需要 GUI 式冲突解决、交互式 rebase 的人（请用 lazygit / TortoiseGit）
- 非 Windows 用户（首版）

---

## 五、核心场景（按优先级）

### S1 忽略管理 ⭐ 最独特，MVP 首选
扫描工作区 → 按「目录名 / 扩展名」聚类候选规则 → **每条规则实时预览「这会忽略 N 个文件、共 X MB」** → 勾选 → 写入 `.gitignore`。
**硬约束：只写 `.gitignore`，绝不移动、删除、改动任何本地文件。**

实现要点（v1.1 评审补，S1 的工程核心所在）：
- **规则匹配用现成 gitignore 语义库**（如 `sabhiram/go-gitignore`），不要基于 `filepath.Match` 手搓——后者不支持 `**`、否定 `!`、`foo/` 仅目录、`/foo` 锚定，也不感知 Windows 的大小写不敏感（`core.ignorecase`）；用 `git check-ignore` 抽样对拍验证
- **候选规则先排除 `.gitignore` 已有规则**，避免重复写入
- **影响预览取增量语义**：N 只统计「新增被忽略」的文件（排除已忽略与已跟踪的），否则数字虚高、误导用户；目录规则命中后不再下钻（与 git 一致：父目录被排除时内部无法重新包含），每条规则的影响只算一次并缓存，勾选切换仅重算对应规则
- **应用后执行 `git ls-files -c -i --exclude-standard`**：命中数 > 0 时提示「N 个已跟踪文件不受 `.gitignore` 影响」——只提示，不代用户执行 `git rm --cached`（留待 v0.2 评估）

内置 **Agent 噪音目录库**（开箱即用，无需用户懂规则）：
```
node_modules/  .venv/  venv/  __pycache__/  .pytest_cache/  .mypy_cache/
.claude/  .cursor/  .codex/  .aider*  .specstory/  .continue/
dist/  build/  out/  target/  *.log  .DS_Store  Thumbs.db  *.tmp
```

### S2 一键提交流
看变更 → 勾选文件 → 写消息（手写或 AI 生成）→ 提交 → 推送 → 建 PR，**全程TUI按钮**。

### S3 提交前风险拦截 ⭐ AI 时代真需求
- **敏感文件**：`.env` / `*apikey*` / `*api_key*` / `*.pem` / `*.key` / `id_rsa` / `*token*` / `*secret*` → 高亮警告，默认不勾选
- **大文件**：> 50MB 警告，> 100MB 阻断（GitHub 硬限制 100MB，本项目强依赖git cli）
- 本质：**把"事故"变成"提示"**

### S4 GitHub 元数据快捷标注
labels / topics / description / homepage —— 本地提交完顺手标掉，不跳浏览器。

### S5 零命令入口（"一次配置"的落点）
- 首次运行向导：检测 git/gh → 引导登录（或复用已有 gh 登录态）→ 选默认编辑器 → 落 config → **注册资源管理器右键菜单**（写 `HKCU\...\Directory\shell\<name>`，**无需管理员**）
- 此后：右键文件夹 →「用 XX 打开」→ TUI 直接起来
- 备选入口：把目录拖进终端窗口、`cd` 后敲一条短命令

### S6 状态一屏直观
本地未提交数 / 未推送数 / 远端未拉取数 / open PR 与 issue 数 —— 顶栏一屏看完。

---

## 六、设计原则（不可妥协）

1. **一次配置**：向导只出现一次，之后任何操作不再追问。config 落 `~/.config/repobuddy/`（或 `%APPDATA%`）
2. **按钮优先**：高频动作 = 看到 → 一个键 → 确认。**每个按钮必须有键盘等价操作**（鼠标在 SSH/老终端下不可用）
3. **不动工作区文件**：**绝不修改/移动/删除任何工作区文件**，唯一例外是 `.gitignore`（S1 的写入目标）；对 git 内部数据（index / objects / refs）与注册表、config 的写操作，仅限用户显式发起的动作（暂存 / 提交 / 推送 / 注册右键菜单）
4. **远端可回退**：所有 GitHub 写操作前记录原值，支持一键撤销
5. **零后端**：不引入任何服务器；凭据不落明文，优先复用 gh 登录态（`gh auth token` / `git credential fill`），绝不要求用户再贴 PAT
6. **不依赖 Nerd Font**：只用 Unicode 制表符与盒线字符，保证任何默认终端都能正确显示
7. **复用而非重写**：git 操作调 `git` CLI，GitHub 操作走 REST/GraphQL，不自己实现 git

---

## 七、架构（四层，UI 层收敛为 TUI）

```
① UI 层 = TUI（唯一前端）
        ↓
② 稳定接口：CLI 子命令 + JSON 输出  ·  MCP server
        ↓
③ core 内核（与 UI 同语言，但接口保持语言无关）
   扫描 / 忽略管理 / 提交推送 / 元数据 / 状态 / 风险检测
        ↓
④ 插件协议（预留，v0.x 不实现）：stdio JSON-RPC（与 MCP 同规范）
```

- **core 与 TUI 用同一门语言起步**（避免混合语言的构建/调试/分发成本）
- 但 core 的对外接口**从第一天就是 CLI + JSON**，保证将来可以：换 UI、加 GUI、被 agent 识别与调用、被脚本调用，都不动核心
- **MCP server 是一等公民入口**：core 暴露 MCP 后，`codex` / `opencode` / `claude` / WorkBuddy 全部直调，无需任何适配 —— 这是可扩展性的主要兑现方式

---

## 八、技术选型（GUI 已排除，只剩 TUI 路线）

| 方案 | GitHub 库 | TUI/CJK | 打包 | 开发速度 | 开源友好 | 结论 |
|---|---|---|---|---|---|---|
| **Go + Bubbletea/Lipgloss** | **官方 go-github** | **强**（go-runewidth 成熟） | **~12MB 单文件** | 快 | **高** | ⭐ **推荐** |
| C# + Terminal.Gui + Octokit | 官方 Octokit | 中（CJK 有历史坑，需实测） | ~70MB（需先装 .NET SDK） | 快 | 高 | 备选 |
| Rust + ratatui | octocrab（非官方） | 强 | ~5MB | **慢** | 中 | 不选（开发太慢） |
| Python + Textual | 官方 PyGithub | 强 | 30-50MB + 启动慢 | **最快** | 高 | 仅用于原型验证 |
| C++20 + FTXUI | **无官方库**（要自己包 REST） | **最难**（双宽处理最不成熟） | 需 vcpkg | 中 | **低** | 出局 |

**为什么 C++ 出局**：GUI 被排除后，C++ 只剩 FTXUI 这条路，而它的三个短板（无官方 GitHub 库、TUI 的 CJK 处理最难、Windows C++ 构建环境对贡献者门槛最高）恰好全部命中，且原本能弥补它的"你熟 Qt"优势在 TUI 里用不上。

**推荐：Go + Bubbletea。**
- Bubbletea 是现有 TUI 框架里最接近"全屏应用"的（组件/鼠标/样式），最接近 lazygit 手感
- `go-github` 官方库 + `golang.org/x/sys/windows`（注册表/DPAPI/文件属性全覆盖）
- 单文件 ~12MB，`GOOS=windows go build` 一条命令出 exe
- 同类参考项目全是 Go：lazygit、froggit、github-tui —— 踩坑经验可抄

**鉴权：零配置复用现有 gh 登录态**
- 读 token：`gh auth token`，或走 git credential helper（本机已配置 `credential.https://github.com.helper = !gh auth git-credential`）
- **不要求用户提供 PAT，不做自己的登录流程** —— 这本身就是"一次配置"的最大简化

---

## 九、TUI 界面草案

```
┌ repo: easy_clean  ⎇ main  ↑0 ↓2  ●3 已改  ●12 未跟踪  ⚠1 风险 ─────────────┐
│ ┌─ 变更 (15) ────────┐ ┌─ 预览 / diff ───────────────────────────────────┐ │
│ │ [x] src/main.cpp   │ │ @@ -12,6 +12,8 @@                               │ │
│ │ [ ] src/util.cpp   │ │ +  foo()                                        │ │
│ │ ⚠ .env             │ │                                                 │ │
│ │ [ ] dist/          │ │                                                 │ │
│ ├─ 候选忽略 (4) ──────┤ │                                                 │ │
│ │ ☑ node_modules/    │ │   会忽略 1,204 个文件，约 340 MB                 │ │
│ │ ☐ .venv/           │ │   会忽略   86 个文件，约  22 MB                  │ │
│ │ ☐ .claude/         │ │                                                 │ │
│ │ ☐ dist/            │ │                                                 │ │
│ └────────────────────┘ └─────────────────────────────────────────────────┘ │
│ [F1]扫描 [F2]应用忽略 [F3]提交 [F4]推送/PR [F5]标签 [F6]风险 [Tab]切换 [?]帮助│
└────────────────────────────────────────────────────────────────────────────┘
```

要点：
- 左侧两栏：**变更列表** + **候选忽略列表**（勾选即生效）
- 右侧：diff 预览 / 规则影响预览（"会忽略 N 个文件、X MB"必须实时算）
- 顶栏：一屏状态（S6）
- 底栏：功能键提示，**每个动作都有键盘等价**
- 风险项（`⚠ .env`）自动带标记且**默认不勾选**

---

## 十、对外接口草案（core 的契约）

### 10.1 CLI 子命令

```
repobuddy                     # 默认进 TUI
repobuddy tui [path]          # 进 TUI
repobuddy scan [path] --json  # 工作区扫描：已暂存/已修改/未跟踪/已忽略 四类（2026-09-28 由三类增补，见 18.11）
repobuddy ignore --suggest    # 输出候选规则 + 影响预览（N 文件 / X MB）
repobuddy ignore --apply -r <规则>...   # 写入 .gitignore（可多选）
repobuddy risk                # 敏感文件 + 大文件扫描
repobuddy commit -m "msg" [--files ...] [--push] [--pr]
repobuddy meta labels [--add|--remove|--list] ...
repobuddy meta topics [--set ...]
repobuddy meta desc [--set "..."]
repobuddy status --json       # 本地 + 远端状态
repobuddy serve-mcp           # 以 MCP server 模式运行（stdio）
repobuddy setup               # 首次向导（含右键菜单注册/卸载）
repobuddy setup --unregister  # 卸载右键菜单
```

所有支持 `--json` 的命令，JSON 结构写进文档并带 schema 版本字段（如 `"schema": 1`） —— 这是"将来换 UI / 加 GUI"的契约。**v1.0 前该契约标记为 unstable**：允许随迭代调整，调整时同步更新文档与 schema 版本号；v1.0 起冻结。

### 10.2 MCP 工具清单

| 工具 | 入参 | 出参 |
|---|---|---|
| `repo_scan` | path | 已暂存/已修改/未跟踪/已忽略 列表（同 10.1，2026-09-28 增补） |
| `ignore_suggest` | path | 候选规则 + 每条的影响（文件数/体积） |
| `ignore_apply` | path, rules[] | 写入结果 + 新 `.gitignore` 内容 |
| `detect_risks` | path | 敏感文件列表 / 超大文件列表 |
| `commit_changes` | path, message, files[], push?, create_pr? | commit sha / PR url |
| `set_labels` | repo, add[], remove[] | 结果 |
| `set_topics` | repo, topics[] | 结果 |
| `repo_status` | path | 分支/领先落后/PR/issue 概览 |

---

## 十一、功能分期

### MVP（目标：跑通一条完整链路；实现顺序 = 下列顺序：core 先行 → TUI 包壳 → setup/右键菜单最后）
- [x] `scan` 工作区扫描（四类分组：已暂存/已修改/未跟踪/已忽略）—— 2026-09-28 完成（commit `a09727e`）：
      人读 + `--json` 两种输出，5 个单元测试通过，契约见 `docs/json-contract.md`
- [ ] `ignore --suggest / --apply`：候选规则 + 实时影响预览（增量语义）+ 写入 `.gitignore`（排除已有规则；应用后提示已跟踪文件不受影响，见 S1 实现要点）
- [ ] `risk` 敏感文件 + 大文件拦截（提交前自动跑）
- [ ] 勾选暂存 + 提交（消息手写）+ 推送
- [ ] TUI 主界面（左侧双栏 + 右侧预览 + 顶栏状态 + 底栏快捷键）
- [ ] `--json` 输出（接口契约）—— 2026-09-28：`scan --json` 已就绪（见 18.11），其余命令待补
- [ ] `setup` 向导：检测 git/gh → 复用登录态 → 落 config → 注册右键菜单（**MVP 序列最后实现**，单个菜单项；MVP 中唯一涉及注册表、对核心差异化零贡献的模块，见风险 #3）

### v0.2
- [ ] `serve-mcp`（MCP server 入口）
- [ ] `meta labels / topics / desc`
- [ ] 一键建 PR（`commit --pr`）
- [ ] 右键菜单子菜单化（避免与 TortoiseGit 菜单打架）

### v0.3+
- [ ] AI 生成提交信息（可插本地模型 / 已有 codex / opencode CLI）
- [ ] 多仓库仪表盘
- [ ] 插件协议（stdio JSON-RPC）正式开放
### 其它
待定：树状文件菜单显示

---

## 十二、非功能需求

| 维度 | 要求 |
|---|---|
| 启动 | TUI 冷启动 < 200ms |
| 体积 | 单文件 exe，< 20MB |
| 网络 | 所有 GitHub 调用尊重限流，批量操作节流；断网时 TUI 仍可用（本地功能不依赖网络） |
| 安全 | 凭据不落明文（复用 gh keyring，token 仅内存持有）；`.gitignore` 写入前显示 diff、内存保留原文、会话内一键撤销（不做临时目录备份） |
| 兼容 | Windows Terminal（主）/ conhost（降级）；**不要求 Nerd Font** |
| 可卸载 | `setup --unregister` 一键清除右键菜单与 config |
| 中文 | 中英混排对齐必须正确（这是 TUI 的经典坑，立项第一周就要验证） |

---

## 十三、风险与坑

1. **中文宽字符对齐** —— 最大技术风险，第一周必须做出对齐验证 demo
2. **鼠标支持不可假设** —— SSH/老终端没有鼠标，所有按钮必须有键盘等价
3. **右键菜单与 TortoiseGit 冲突** —— MVP 只注册单个菜单项（子菜单化在 v0.2）；注册表传参：右键文件夹图标走 `Directory\shell`（用 %1），右键文件夹空白处走 `Directory\Background\shell`（用 %V），勿混用
4. **写 HKCU 注册表可能被杀软误报** —— 准备说明文档；提供"不装右键菜单"的纯命令行模式
5. **GitHub API 限流** —— 批量 labels/topics 要节流；多文件提交是多次调用，需要失败回滚（partial commit 很脏）
6. **git CLI 依赖** —— 用户没装 git 时向导要能引导（用 winget 装）
7. **平台寄生** —— GitHub 改版/API 变动随时可能破坏功能；尽量依赖稳定的 REST/GraphQL 而非网页 DOM
8. **官方补位风险** —— 别做官方明天就能做的功能，专注"官方没动力做的长尾整合"

---

## 十四、里程碑

| 版本 | 内容 | 验收标准 |
|---|---|---|
| **v0.1 MVP** | scan + ignore + risk + commit/push + TUI + setup/右键菜单（最后） | 在 easy_clean 上完成一次"扫描→忽略→提交→推送"；已注册右键菜单时全程零命令输入，跳过右键菜单时允许「打开终端输入一条启动命令」 |
| **v0.2** | MCP server + 元数据 + 一键 PR | codex/opencode 能通过 MCP 直接完成"扫描+忽略" |
| **v0.3** | AI 提交信息 + 多仓库仪表盘 | — |
| **v1.0** | 插件协议开放 + 打磨 | 首个外部贡献者 PR 能顺利合入 |

---

## 十五、待拍板

1. **技术栈**：Go + Bubbletea（推荐）还是 C# + Terminal.Gui？
2. **MVP 切入点**：S1 忽略管理（推荐，无竞品、demo 直观）还是 S2 一键提交流？
3. **项目名**：暂定 `repo-buddy`（中文可叫「仓库管家」），备选 `gitdock` / `repotidy`
4. **项目路径**：建议 `E:\`（空间最充裕 207.7GB，easy_clean 也在 E:）
5. **右键菜单**：是否进入 MVP？（涉及写 HKCU 注册表）

---

## 十六、基础配置（2026-09-22 增补）

> 本节起状态：**已拍板**，覆盖第 15 节待拍板项 ——
> ① 技术栈 = **Go + Bubbletea** ② MVP = **S1 忽略管理** ③ 项目名 = **repo-buddy**（仓库管家）
> ④ 项目路径 = **`E:\repo-buddy`** ⑤ 右键菜单 **进 MVP，但 setup 时可跳过**（保留纯命令行模式）

### 16.1 环境前提（2026-09-22 实测）

| 项 | 状态 |
|---|---|
| git 2.47.1（凭据复用 gh） | ✅ |
| gh 2.99.0（已登录 zmw-code） | ✅ |
| Windows Terminal 1.24 / 长路径已开 | ✅ |
| **Go** | ✅ go1.27.1（`C:\Program Files\Go`，2026-09-28 实测；原记「未安装」已过时） |
| **GOPROXY** | ✅ `https://goproxy.cn,direct`（2026-09-28 实测） |
| PowerShell | 仅 5.1（脚本不能用 `??`/三元） |

> 2026-09-28 更新：上表 Go 与 GOPROXY 两行原为「未安装 / 为空」，实测已就绪，
> 故 16.2 的安装步骤对当前机器已无需执行（保留作他人环境参考）。

### 16.2 安装 Go（二选一）

**A. winget（需管理员，一次到位）**

```powershell
winget install --id GoLang.Go -e
```

**B. 绿色版（免管理员，与 w64devkit 同级放 `E:\devkit`，推荐）**

```powershell
# 1. 先去 https://go.dev/dl/ 确认最新 stable 版本号，替换 $ver
$ver  = "1.23.4"
$dest = "E:\devkit"
Invoke-WebRequest "https://go.dev/dl/go$ver.windows-amd64.zip" -OutFile "$env:TEMP\go.zip"
Expand-Archive "$env:TEMP\go.zip" -DestinationPath $dest -Force

# 2. 写用户级环境变量（持久，无需管理员）
[Environment]::SetEnvironmentVariable("GOROOT", "$dest\go",   "User")
[Environment]::SetEnvironmentVariable("GOPATH", "E:\go-work", "User")
$p = [Environment]::GetEnvironmentVariable("Path", "User")
if ($p -notlike "*$dest\go\bin*") {
  [Environment]::SetEnvironmentVariable("Path", "$p;$dest\go\bin;E:\go-work\bin", "User")
}

# 3. 国内镜像（关键）
[Environment]::SetEnvironmentVariable("GOPROXY", "https://goproxy.cn,direct", "User")
```

> 安全提示：只改 `GOPROXY`，**不要关 `GOSUMDB`** —— goproxy.cn 支持代理校验库，安全性不降级。

**验证（重开终端后）**

```powershell
go version        # 应输出 go1.x
go env GOPROXY    # 应输出 https://goproxy.cn,direct
```

### 16.3 项目骨架（已建）

```
E:\repo-buddy\
├─ TUI仓库管理工具-需求分析.md   # 需求（权威版本）
├─ README.md
├─ .gitignore                    # 直接采用 S1 的 agent 噪音库，自食其狗粮
├─ go.mod                        # module github.com/zmw-code/repo-buddy
├─ main.go                       # 最小可编译骨架（子命令占位，纯标准库零依赖）
├─ cmd/                          # CLI 子命令实现（对应 10.1 节契约）
├─ internal/core/                # 内核：扫描/忽略/提交/元数据/风险（只暴露接口）
├─ internal/tui/                 # Bubbletea 界面（第九节草案）
├─ internal/mcp/                 # serve-mcp（v0.2）
└─ docs/                         # 设计与决策记录
```

骨架验收（Go 装好后）：

```powershell
cd E:\repo-buddy
go run . version    # 应输出 repo-buddy 0.0.1-dev
go run . help       # 应打印子命令清单
```

---

## 十七、开发工作流（2026-09-22 增补）

### 17.1 分支与提交

- 分支：`main` + `feat/*`；**`main` 必须保持可编译可运行**
- 提交信息：Conventional Commits（`feat:` / `fix:` / `docs:` / `chore:`）—— 为将来自动生成 changelog
- 提交前必跑：`go build ./...` + `go vet ./...`

### 17.2 日常循环

```
改代码 → go build ./... → go run . scan（手验）→ go vet ./... → 提交
```

### 17.3 agent 协作流（本项目主工作流）

1. **任务来源 = 第 10 节接口契约**。每个 CLI 子命令 / MCP 工具就是一张任务单，做完打勾
2. 实现可派给 codex / opencode：把「对应契约条目 + 涉及文件」喂给它，产出补丁
3. WorkBuddy 负责：任务拆解、契约评审、代码验收、跑验收标准
4. **验收标准以第 14 节里程碑为准**：v0.1 = 在 easy_clean 上"扫描→忽略→提交→推送"（零命令输入口径见第 14 节 v0.1 行，含右键菜单跳过时的例外）

### 17.4 发布

```powershell
go build -trimpath -ldflags "-s -w" -o repobuddy.exe .
```

### 17.5 首次发布 checklist

- [x] 装 Go + 配 GOPROXY（16.2）—— 2026-09-28 实测：go1.27.1 + `goproxy.cn`
- [x] `go run . version` 通过（16.3）—— 输出 `repo-buddy 0.0.1-dev`
- [x] **中文宽字符对齐验证 demo**（风险 #1，第一周必做）—— `repobuddy cjk-check`
      （自动断言 11 项：宽度模型 / 盒线闭合 / 表格列对齐 / 按宽度截断；终端与字体的
      目视确认在该命令第 4 节，换字体或字号后需人工重跑一次）
- [x] `gh repo create repo-buddy --public --source . --push`
      —— https://github.com/zmw-code/repo-buddy （2026-09-28）
- [x] 加 LICENSE（建议 MIT，与 easy_clean 对齐）+ 首个 commit
      —— MIT（`LICENSE`）；首个 commit `00c09ce`

---

## 十八、评审修订记录（2026-09-27 增补）

> 2026-09-27 评审结论：需求成立、选型正确，可开工。随评审应用的修订均已直接写入正文，本节按「原文 → 修改后」对照保留原文，便于回溯。
> 修订原则：**遵循 MVP，只修会误导实现（本文档要直接喂给 coding agent）或影响 S1 可信度的问题**；不新增功能，不重开已拍板决定。

### 18.1 设计原则 3：消除与 S2 的字面矛盾 ⭐ 最重要

| 对照 | 内容 |
|---|---|
| 原文 | 3. **本地只读承诺**：除了 `.gitignore` 这一个文件，**绝不修改/移动/删除任何本地文件** |
| 修改后 | 3. **不动工作区文件**：**绝不修改/移动/删除任何工作区文件**，唯一例外是 `.gitignore`（S1 的写入目标）；对 git 内部数据（index / objects / refs）与注册表、config 的写操作，仅限用户显式发起的动作（暂存 / 提交 / 推送 / 注册右键菜单） |

理由：S2 的 `git add` 写 index、`git commit` 写 objects/refs，按原文严格字面即违规；文档自称「可直接作为 coding agent 的输入」，较真的 agent 会在实现 `git add` 时卡壳。保护对象（用户工作区文件）一个没少，只是把绝对化表述改精确。

### 18.2 S1 新增「实现要点」（原文无对应内容）

原文 S1 只有场景描述，缺工程核心。新增四条（正文见第五节 S1）：

1. 规则匹配用现成 gitignore 语义库 + `git check-ignore` 抽样对拍（`filepath.Match` 不支持 `**` / `!` / `foo/` / `/foo` 锚定 / Windows 大小写语义）
2. 候选规则先排除 `.gitignore` 已有规则，避免重复写入
3. 影响预览取**增量语义**（排除已忽略与已跟踪的文件），目录规则命中即停、按规则缓存
4. 应用后跑 `git ls-files -c -i --exclude-standard`，提示「N 个已跟踪文件不受影响」——只提示不代做

理由：1、3 决定预览数字是否可信、行为是否与 git 一致（S1 的立身之本）；2 防重复写入；4 补上「`.gitignore` 不会 untrack 已跟踪文件」这一新手最大困惑，且是只读操作，不违反原则 3。

### 18.3 MVP 清单重排：右键菜单挪到最后

| 对照 | 顺序 |
|---|---|
| 原文 | setup 向导 → scan → ignore → commit/push → risk → TUI → --json |
| 修改后 | scan → ignore → risk → commit/push → TUI → --json → setup 向导 + 右键菜单（最后，单个菜单项） |

理由：右键菜单是 MVP 中唯一碰注册表、唯一有杀软误报风险、对核心差异化零贡献的模块，编号 S5 容易诱导先做它；core 先行使每步都有可独立验收的产物（如 `go run . scan --json`），与 17.3「契约条目即任务单」的节奏一致。

### 18.4 里程碑 v0.1 验收：与可跳过的右键菜单解耦

| 对照 | 内容 |
|---|---|
| 原文 | 在 easy_clean 上完成一次"扫描→忽略→提交→推送"全程零命令输入 |
| 修改后 | 在 easy_clean 上完成一次"扫描→忽略→提交→推送"；已注册右键菜单时全程零命令输入，跳过右键菜单时允许「打开终端输入一条启动命令」 |

理由：第十六节拍板「右键菜单 setup 时可跳过」，而备选入口（拖进终端 / cd 后敲命令）本身含命令输入——原验收在跳过场景下不可能达成。17.3 第 4 条的转述同步更新。

### 18.5 JSON 契约：加 schema 字段 + 冻结期限

| 对照 | 内容 |
|---|---|
| 原文 | JSON 结构必须稳定并写进文档 |
| 修改后 | JSON 结构写进文档并带 schema 版本字段（如 `"schema": 1`）；v1.0 前标记 unstable，随迭代调整并同步文档与版本号，v1.0 起冻结 |

理由：契约当任务单很好（17.3），当 0.x 阶段的兼容承诺太早——写太满会拖累迭代速度。

### 18.6 非功能-安全：`.gitignore` 保护机制简化

| 对照 | 内容 |
|---|---|
| 原文 | 凭据不落明文（复用 gh keyring）；`.gitignore` 写入前备份原文件到临时目录 |
| 修改后 | 凭据不落明文（复用 gh keyring，token 仅内存持有）；`.gitignore` 写入前显示 diff、内存保留原文、会话内一键撤销（不做临时目录备份） |

理由：diff + 会话内一键撤销更符合直觉、无临时文件残留；原文方案无害，属低成本替换，非必须。

### 18.7 风险 #3：右键菜单实现细节补充

| 对照 | 内容 |
|---|---|
| 原文 | 3. **右键菜单与 TortoiseGit 冲突** —— 用单个子菜单收纳 |
| 修改后 | 3. **右键菜单与 TortoiseGit 冲突** —— MVP 只注册单个菜单项（子菜单化在 v0.2）；注册表传参：右键文件夹图标走 `Directory\shell`（用 %1），右键文件夹空白处走 `Directory\Background\shell`（用 %V），勿混用 |

理由：原文「用单个子菜单收纳」与「MVP 单菜单项 / v0.2 子菜单化」的分期矛盾；`%1` 与 `%V` 是注册表经典 gotcha，实现时必撞，提前记录。

### 18.8 文档头状态行（原本已过时）

| 对照 | 内容 |
|---|---|
| 原文 | 版本：v1.0（2026-09-22）；状态：待拍板（技术栈二选一 / MVP 切入点 / 项目名） |
| 修改后 | 版本：v1.1（2026-09-27）；状态：已拍板（第十六节）；评审修订对照见第十八节 |

理由：第十六节早已拍板，状态行停留在拍板前，顺带更新。

### 18.9 README 同步（两处）

1. 设计底线第 1 条随 18.1 同步改写；
2. 「见需求文档 16.5 节 checklist」→「17.5 节」——原文引用的 16.5 不存在，LICENSE 建议与 checklist 实际在 17.5。

### 18.10 评审看过、明确不改的

- 技术栈（Go + Bubbletea）、鉴权方案（复用 gh 登录态）、零后端红线 —— 均正确，不动
- 右键菜单进 MVP 的拍板 —— 维持，仅调实现顺序（18.3）
- MVP 功能组成 —— 零增删，仅调顺序与标注（18.3）
- CJK 对齐列为第一周验证 —— 原文已正确，不动
- Go 版本示例（1.23.x）—— 16.2 本就要求装前查最新版，不改文档

### 18.11 `scan` 输出由三类改为四类（2026-09-28 增补，需求实现反馈）

| 原文 | `scan` 输出「未跟踪 / 已修改 / 已忽略 三类」（10.1、10.2、11） |
| 修改后 | 「已暂存 / 已修改 / 未跟踪 / 已忽略 四类」（10.1、10.2、11） |

理由：`git status --porcelain` 的两位状态码，**首列本来就是「index 有改动」与「仅工作区有改动」的区分**
（`M ` vs ` M`）。按三类实现等于把这条信息丢掉，而 MVP 第 4 项（勾选暂存 + 提交）与 TUI 列表
必须区分二者，丢掉就要在 TUI 阶段返工；「哪些已暂存」也正是提交流程第一步要展示的东西。

性质：**增补，不是破坏性变更** —— 原有三类的口径与字段都不变，只多一个 `staged` 数组，
`counts` 同步多一个字段；`ignored` 的折叠语义不变。JSON 契约仍为 `schema: 1`（unstable，见 18.5），
分类口径的完整对照表写在 `docs/json-contract.md`。

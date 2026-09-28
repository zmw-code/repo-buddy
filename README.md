# repo-buddy（仓库管家 · 暂定名）

本地 TUI 仓库管理工具：把「本地文件现状」和「GitHub 仓库设置」放进同一个终端界面，
用按钮而不是命令完成日常仓库管理。**一次配置，此后右键 / 简单命令行 / .exe 即用。**

> 形态：**TUI only，不做任何 GUI** ｜ 平台：Windows 优先 ｜ 后端：**零后端**
> 需求文档：[TUI仓库管理工具-需求分析.md](./TUI仓库管理工具-需求分析.md)（权威版本）

## 它解决什么

- agent 一口气生成几十个文件，分不清哪些该提交、哪些该忽略 → **忽略管理**（含实时影响预览）
- agent 容易把密钥写进代码 → **提交前风险拦截**（敏感文件 / 大文件）
- GitHub 的 labels / topics / description 只能在网页里点 → **元数据快捷标注**
- 提交要敲一串命令 → **一键提交流**

## 状态

`v0.0.1-dev` — 骨架已就位：子命令分发（`cmd/`）+ `scan` 工作区扫描 + 中文宽字符对齐自检（通过）；
MVP 其余功能（`ignore` / `risk` / 提交推送 / TUI / `setup`）尚未开始。
远端仓库：[zmw-code/repo-buddy](https://github.com/zmw-code/repo-buddy)

## 用法

```powershell
go run . scan              # 工作区扫描：已暂存 / 已修改 / 未跟踪 / 已忽略
go run . scan --json       # 同上，机器可读（结构见 docs/json-contract.md）
go run . cjk-check         # 中文宽字符对齐自检（需求文档 风险 #1）
```

## 构建

```powershell
go build -trimpath -ldflags "-s -w" -o repobuddy.exe .
```

## 自检

```powershell
go run . cjk-check    # 中文宽字符对齐自检（需求文档 风险 #1）
```

输出 11 项自动断言（宽度模型 / 盒线闭合 / 表格列对齐 / 按宽度截断）+ 一节供人眼确认的
目视材料；换字体、改字号或缩放窗口后建议重跑。退出码 0 = 通过。

## 设计底线（不可妥协）

1. 不动工作区文件：除 `.gitignore` 外，绝不修改 / 移动 / 删除任何工作区文件；对 git 内部数据的写操作仅限用户显式发起的暂存 / 提交 / 推送（口径见需求文档 18.1）
2. 零后端：不引入任何服务器，凭据复用 `gh` 登录态
3. 不依赖 Nerd Font：只用 Unicode 制表符与盒线字符
4. 每个按钮都有键盘等价操作

## 协议

MIT，见 [LICENSE](./LICENSE)。

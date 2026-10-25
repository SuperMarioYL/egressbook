[English](./README.md) | **简体中文**

<div align="center">

<img src="https://readme-typing-svg.demolab.com?font=Fira+Code&weight=600&size=28&duration=3000&pause=1000&color=58A6FF&center=true&vCenter=true&width=620&height=100&lines=egressbook;编码+Agent+会话;出站披露账本" alt="egressbook" />

<br/>

<a href="https://github.com/SuperMarioYL/egressbook/actions"><img src="https://img.shields.io/github/actions/workflow/status/SuperMarioYL/egressbook/ci.yml?style=flat-square&label=CI" alt="CI" /></a>
<a href="https://github.com/SuperMarioYL/egressbook/releases"><img src="https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go" /></a>
<a href="https://github.com/SuperMarioYL/egressbook/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-58A6FF?style=flat-square" alt="License" /></a>
<a href="https://github.com/SuperMarioYL/egressbook/releases"><img src="https://img.shields.io/badge/version-0.2.0-58A6FF?style=flat-square" alt="Version" /></a>

<br/>
<br/>

**你的编码 Agent 已经跑了几个月。两篇头版披露刚刚证实其中一个在静默上传 Git 历史——已经离开你机器的，究竟是什么、去了哪？**

egressbook 用你磁盘上本来就有的会话日志回答这个问题：一份**签名的、按目的地映射的披露账本**——Agent 发出去的每个文件、代码片段、系统提示词、工具结果，全部证据可回溯到原始日志行。

免费 · 开源 · MIT 协议 · **零网络请求——只读本地日志**

</div>

---

## 目录

- [为什么需要 egressbook](#为什么需要-egressbook)
- [快速开始](#快速开始)
- [命令一览](#命令一览)
- [实录演示](#实录演示)
- [架构](#架构)
- [覆盖度诚实声明](#覆盖度诚实声明)
- [路线图](#路线图)

---

## 为什么需要 egressbook

2026 年 9 月，两位独立研究者分别证实了一个广泛使用的编码 Agent 在静默上传 Git 历史与工作区快照到厂商云端（[披露一](https://tokenstead.ai/guides/zcode-silent-git-history-upload)，Hacker News 257 分；[披露二](https://blog.ferstar.org/en/posts/zcode-silent-workspace-snapshot-upload/)，205 分）。发现来自外部研究者的手工取证——因为没有任何工具把你机器上已有的日志当作出站记录看待。

现有方案在时间方向上都指错了：

| 方案 | 看得见什么 | 错过什么 |
|---|---|---|
| 出站绊线 / 蜜罐 | **下一次**未授权连接 | 安装之前发出的一切 |
| 脱敏网关 | 配置**之后**流经它的流量 | 你没接管的厂商遥测通道；所有历史会话 |
| 信任厂商 | 厂商的透明度页面 | 披露事件本身——厂商永远最后知道 |

egressbook 指向**过去**：解析 ZCode、Codex、Claude Code 的本地会话日志——包括 ZCode 写在 `~/.zcode/cli/rollout/` 下的完整出站请求体——重建「什么内容、发给了哪个厂商、什么时候、多少字节」。产物不是截图也不是口头保证：是一份**签名账本**，每一项都带回到精确日志行的证据定位符，审查者可以逐条核验。

这是一个新的 agent 流程原语——**会话出站披露账本**：可跨会话 diff 的可审计资产，而不是一块盯着看的仪表盘。

## 快速开始

```bash
go install github.com/SuperMarioYL/egressbook@latest

# 1. 看看这台机器上有哪些 Agent 日志
egressbook scan

# 2. 生成披露账本（签名、按目的地映射）
egressbook ledger

# 3. 导出可分享的 HTML 报告
egressbook report --html egressbook-report.html

# 4. 最近两周「什么离开了机器」有什么变化
egressbook diff --since 2w
```

无配置文件、无账号、无 API key。二进制零网络连接——审计出站的工具自己不能联网，你可以用防火墙验证这一点。

## 命令一览

| 命令 | 作用 |
|---|---|
| `egressbook scan` | 探测 ZCode / Codex / Claude Code 会话日志，报告每 harness 覆盖度与诚实的缺口 |
| `egressbook ledger [--out ledger.json]` | 构建并签名披露账本；打印目的地映射与载荷类型统计 |
| `egressbook report --html out.html` | 渲染自包含、无脚本的 HTML 报告（可脱敏分享的工件） |
| `egressbook diff --since 2w` \| `--old a.json --new b.json` | 「什么离开了机器」的变化：新增目的地、新增载荷类型、按目的地的增量 |
| `egressbook verify ledger.json` | 校验账本的 ed25519 签名——签名后任何改动都会被发现 |

## 实录演示

下方演示运行在仓库自带的自包含样本环境（`testdata/samples/`，内容已脱敏）——同一场景记录于 [`docs/demo-results.json`](./docs/demo-results.json)：

- 探测到 3 个 harness、解析 3 个会话、11 个出站条目
- 目的地映射：Zhipu (GLM) 240 B · OpenAI 154 B · Anthropic (Claude) 63 B
- `ed25519` 签名校验通过；篡改任意条目后 `verify` 失败

`.cast` 录像在 [`assets/demo.cast`](./assets/demo.cast)——本地回放：`asciinema play assets/demo.cast`。

## 架构

```
~/.zcode/cli/rollout/ ─┐
~/.codex/sessions/     ├─→ ingest（3 个解析器）─→ analyze（分类 + 目的地映射）
~/.claude/projects/   ─┘                            │
                                                    ▼
                                    model.Ledger（规范化 JSON + ed25519 签名）
                                       │            │            │
                                   report.Term    report.HTML   report.Diff
```

- **ingest** — 每个 harness 日志格式一个解析器；输出带「文件+行号」证据定位符的规范化事件
- **analyze** — 载荷分类（`system_prompt` / `user_message` / `file_content` / `code_snippet` / `tool_result` / `git_artifact`）+ 目的地归一；不透明 provider id 回退到模型厂商名
- **model** — 账本：规范化序列化、持久 ed25519 密钥（`~/.egressbook/egressbook.ed25519`，首次运行创建）、校验
- **report** — 终端表格、自包含 HTML、跨账本 diff

单二进制、三个内部层、无服务、无网络。

## 覆盖度诚实声明

披露工具绝不能夸大。egressbook 按 harness 标注日志能证明与不能证明的边界：

- **ZCode**：出站请求体（系统提示词、消息）完整记录在本地——重建基底最强
- **Codex**：会话元数据（厂商、基础指令）、用户轮次、工具输出有记录
- **Claude Code**：用户轮次与工具结果有记录；系统提示词在 harness 侧、每会话厂商**不在**日志里——egressbook 将这些打印为显式缺口，而不是猜测

「已记录的一切」如实标注，绝不冒充「已发送的一切」。

## 路线图

- [ ] v0.2 — Cursor `state.vscdb` 解析器；跨机器的签名导出持久身份
- [ ] v0.2 — `--redact` HTML 导出模式（路径/主机名脱敏，便于分享）
- [ ] v0.3 — 策略 diff 包：面向安全审查周期的签名「变化」工件
- [ ] 欢迎提想法——开个 issue

<div align="center">

<sub><a href="./LICENSE">MIT</a> © 2026 SuperMarioYL</sub>

</div>

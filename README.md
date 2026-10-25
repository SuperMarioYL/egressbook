**English** | [简体中文](./README.zh-CN.md)

<div align="center">

<img src="https://readme-typing-svg.demolab.com?font=Fira+Code&weight=600&size=28&duration=3000&pause=1000&color=58A6FF&center=true&vCenter=true&width=620&height=100&lines=egressbook;session+egress+disclosure;ledger+for+coding+agents" alt="egressbook" />

<br/>

<a href="https://github.com/SuperMarioYL/egressbook/actions"><img src="https://img.shields.io/github/actions/workflow/status/SuperMarioYL/egressbook/ci.yml?style=flat-square&label=CI" alt="CI" /></a>
<a href="https://github.com/SuperMarioYL/egressbook/releases"><img src="https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go" /></a>
<a href="https://github.com/SuperMarioYL/egressbook/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-58A6FF?style=flat-square" alt="License" /></a>
<a href="https://github.com/SuperMarioYL/egressbook/releases"><img src="https://img.shields.io/badge/version-0.2.0-58A6FF?style=flat-square" alt="Version" /></a>

<br/>
<br/>

**Your coding agent has been running for months. Two front-page disclosures just showed one of them silently uploading Git history. What of yours already left the machine — and to whom?**

egressbook answers that from the session logs already on your disk: a **signed, destination-mapped disclosure ledger** of every file, snippet, system prompt, and tool result your agents transmitted — evidence-linked back to the raw log lines.

Free · Open source · MIT license · **Zero network calls — it reads only local logs**

</div>

---

## Contents

- [Why egressbook](#why-egressbook)
- [Quickstart](#quickstart)
- [Commands](#commands)
- [Recorded demo](#recorded-demo)
- [Architecture](#architecture)
- [Coverage honesty](#coverage-honesty)
- [Roadmap](#roadmap)

---

## Why egressbook

In September 2026, two independent researchers documented a widely-used coding agent silently uploading Git history and workspace snapshots to its vendor's cloud ([disclosure 1](https://tokenstead.ai/guides/zcode-silent-git-history-upload), 257 points on Hacker News; [disclosure 2](https://blog.ferstar.org/en/posts/zcode-silent-workspace-snapshot-upload/), 205 points). Discovery came from outsiders doing forensics by hand — because no tool treats the artifacts already on your machine as an egress record.

The existing answers all point the wrong direction in time:

| Approach | What it sees | What it misses |
|---|---|---|
| Egress tripwires / honeytokens | The **next** unauthorized connection | Everything already sent before you installed them |
| Redaction gateways | Traffic routed through them **after** setup | Vendor telemetry channels you didn't route; all past sessions |
| Trusting the vendor | Their transparency page | The disclosure itself — vendors found out last |

egressbook points **backwards**: it parses the local session logs of ZCode, Codex, and Claude Code — including the full outbound request bodies ZCode writes under `~/.zcode/cli/rollout/` — and reconstructs what was transmitted, to which provider, when, with how many bytes. The output is not a screenshot and not a promise: it is a **signed ledger** in which every item carries an evidence locator back to the exact log line, so a reviewer can verify your claims.

This is a new agent-flow primitive — the **session-egress-disclosure ledger**: an auditable asset you can diff across sessions, not a dashboard you stare at.

## Quickstart

```bash
go install github.com/SuperMarioYL/egressbook@latest

# 1. See what agent logs exist on this machine
egressbook scan

# 2. Build the disclosure ledger (signed, destination-mapped)
egressbook ledger

# 3. Export the shareable HTML report
egressbook report --html egressbook-report.html

# 4. What changed in what leaves, last two weeks
egressbook diff --since 2w
```

No config file, no account, no API key. The binary opens zero network connections — audit a tool that audits egress, and you can prove it with a firewall.

## Commands

| Command | What it does |
|---|---|
| `egressbook scan` | Detect ZCode / Codex / Claude Code session logs and report per-harness coverage + honest gaps |
| `egressbook ledger [--out ledger.json]` | Build and sign the disclosure ledger; print the destination map and payload-kind totals |
| `egressbook report --html out.html` | Render the self-contained, script-free HTML report (shareable redaction-safe artifact) |
| `egressbook diff --since 2w` \| `--old a.json --new b.json` | What changed in what leaves: new destinations, new payload kinds, per-destination deltas |
| `egressbook verify ledger.json` | Verify the ledger's ed25519 signature — catches any edit after signing |

## Recorded demo

The demo below runs against the repo's self-contained sample home (`testdata/samples/`, redacted content) — the same scenario recorded in [`docs/demo-results.json`](./docs/demo-results.json):

- 3 harnesses detected, 3 sessions parsed, 11 egress items
- Destination map: Zhipu (GLM) 240 B · OpenAI 154 B · Anthropic (Claude) 63 B
- Signature `ed25519` verifies; tampering with any item fails `verify`

The `.cast` file lives at [`assets/demo.cast`](./assets/demo.cast) — replay it locally with `asciinema play assets/demo.cast`.

## Architecture

```
~/.zcode/cli/rollout/ ─┐
~/.codex/sessions/     ├─→ ingest (3 parsers) ─→ analyze (classify + dest-map)
~/.claude/projects/   ─┘                            │
                                                    ▼
                                    model.Ledger (canonical JSON + ed25519)
                                       │            │            │
                                   report.Term    report.HTML   report.Diff
```

- **ingest** — one parser per harness log format; emits normalized events with file+line evidence locators
- **analyze** — payload classification (`system_prompt` / `user_message` / `file_content` / `code_snippet` / `tool_result` / `git_artifact`) + provider normalization; opaque provider ids fall back to the model's vendor
- **model** — the ledger: canonical serialization, persistent ed25519 key (`~/.egressbook/egressbook.ed25519`, created on first run), verify
- **report** — terminal table, self-contained HTML, cross-ledger diff

Single binary, three internal layers, no services, no network.

## Coverage honesty

A disclosure tool must never overclaim. egressbook labels what the logs can and cannot prove, per harness:

- **ZCode**: outbound request bodies (system prompts, messages) are recorded locally — the strongest reconstruction substrate
- **Codex**: session metadata (provider, base instructions), user turns, and tool outputs are recorded
- **Claude Code**: user turns and tool results are recorded; the system prompt lives harness-side and the per-session provider is **not** in the log — egressbook prints these as explicit gaps instead of guessing

"Everything logged" is labeled as such. It is never presented as "everything sent."

## Roadmap

- [ ] v0.2 — Cursor `state.vscdb` parser; persistent identity for signed exports across machines
- [ ] v0.2 — `--redact` HTML export mode (path/hostname scrubbing for sharing)
- [ ] v0.3 — policy diff packs: signed "what changed" artifacts for security review cycles
- [ ] ideas welcome — open an issue

<div align="center">

<sub><a href="./LICENSE">MIT</a> © 2026 SuperMarioYL</sub>

</div>

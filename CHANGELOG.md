# Changelog

## 0.1.0 — 2026-09-19

First release: the session-egress-disclosure ledger.

- `scan` — detect ZCode / Codex / Claude Code session logs with per-harness
  coverage and honest gap labeling
- `ledger` — signed (ed25519) destination-mapped disclosure ledger; every item
  carries an evidence locator back to the raw log line
- `report --html` — self-contained, script-free shareable report
- `diff` — cross-ledger or time-split comparison of what changed in what leaves
- `verify` — signature check; any post-signing edit fails
- Payload kinds: system_prompt / user_message / file_content / code_snippet /
  tool_result / git_artifact

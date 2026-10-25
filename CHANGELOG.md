# Changelog

## 0.2.0 — 2026-10-26

Correctness release: five defects found by reading the shipped v0.1.0 source,
each reproduced against a synthetic home.

- `scan` / `ledger` now report a harness whose log directory is absent as an
  explicit coverage gap instead of a bare `found=0` — discovery gaps were being
  computed and then discarded
- `diff --since` rejects zero, negative and overflowing durations instead of
  printing a confident "every destination is gone" report
- the HTML report's evidence locator names the log file again (it was returning
  a directory component, so every row pointed at a shared directory)
- a real git diff is classified `git_artifact` again (the path-header heuristic
  was winning over the `diff --git` header, under-reporting the one payload kind
  the product exists for)
- every user-visible version surface is tied to `VERSION` by a lockstep test

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

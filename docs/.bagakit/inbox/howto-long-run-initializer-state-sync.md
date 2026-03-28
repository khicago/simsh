---
title: Long-run initializer should resync generated state after selecting an item
kind: howto
status: inbox
tags:
  - howto
  - long-run
  - initializer
  - execution-table
sources:
  - .bagakit/long-run/bk-execution-table.json
  - .bagakit/long-run/feature-list.json
  - .bagakit/long-run/next-action.json
  - .bagakit/long-run/bk-execution-handoff.md
created: 2026-03-24T02:39:46Z
---

## Candidate
- Context:
  - During the 2026-03-24 initializer pass for `manual-20260324-agentfs-pathguard-direct-tests`, the pre-session `check_and_resume.sh` run selected the correct row but left it as `todo`, because initializer state had not been written back yet.
  - `feature-list.json` and `next-action.json` are generated from the execution table, so they can drift from the hand-authored handoff if the initializer only edits `bk-execution-handoff.md`.
- How to apply:
  - After choosing the single execution item, set that row to `in_progress` in `.bagakit/long-run/bk-execution-table.json`.
  - Rewrite `.bagakit/long-run/bk-execution-handoff.md` for the same item.
  - Re-run `bash .bagakit/long-run/check_and_resume.sh` once so it resyncs `.bagakit/long-run/feature-list.json` and `.bagakit/long-run/next-action.json` from the updated table.
- Why it matters:
  - This keeps all long-run artifacts pointing at the same current item.
  - It also guarantees `feature-list.json` has exactly one active `in_progress` entry after initializer selection instead of relying on a stale generated snapshot.
- Scope:
  - Applies to initializer passes that move a manual row from `todo` to active work.
  - If the selected row is blocked instead, the same rule still applies: write the status into the table first, then resync generated artifacts.

## Promote To
- `docs/.bagakit/memory/howto-long-run-initializer-state-sync.md` (curated), or
- `docs/<type>-<topic>.md` (normative/deep guide)

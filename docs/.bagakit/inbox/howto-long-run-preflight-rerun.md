---
title: Initializer should rerun long-run preflight before blocking on transient temp-file failures
kind: howto
status: inbox
tags:
  - howto
  - long-run
  - initializer
  - preflight
sources:
  - .bagakit/long-run/check_and_resume.sh
  - .bagakit/long-run/bk-execution-table.json
  - .bagakit/long-run/next-action.json
  - /Users/bytedance/proj/priv/bagakit/skills/dist_local/bagakit-long-run/scripts/validate-long-run.sh
created: 2026-03-24T04:36:12Z
---

## Candidate
- Context:
  - During the 2026-03-24 initializer pass for `manual-20260324-agentfs-mutation-write-limit-regressions`, the first `bash .bagakit/long-run/check_and_resume.sh` run failed inside `validate-long-run.sh` with `cannot create temp file for here document: No space left on device`.
  - A direct `mktemp` check and a second `check_and_resume.sh` run both succeeded a few minutes later without any code or workspace change, and the second run selected the same execution row cleanly.
- How to apply:
  - If initializer preflight fails on a temp-file or low-space error, do one quick environment sanity check first (`mktemp`, or an equivalent tiny write in the repo) before marking the row blocked.
  - Re-run `bash .bagakit/long-run/check_and_resume.sh` once after that sanity check.
  - Only mark the item blocked if the second run still fails, or if the sanity check itself proves the workspace cannot create temporary files.
- Why it matters:
  - A single transient preflight failure can otherwise misclassify an actionable row as blocked even though the execution table and generated state are healthy.
  - The rerun also confirms whether the failure is environmental noise or a durable workspace issue that the handoff must explicitly encode.
- Scope:
  - Applies to initializer or coding passes that rely on `check_and_resume.sh` as the resume gate.
  - Does not replace a real blocked status when temp-file creation is consistently broken.

## Promote To
- `docs/.bagakit/memory/howto-long-run-preflight-rerun.md` (curated), or
- `docs/<type>-<topic>.md` (normative/deep guide)

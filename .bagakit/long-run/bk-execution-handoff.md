# BK Execution Handoff

## Run Metadata

- Updated At (UTC): 2026-04-03T18:20:00Z
- Updated By: codex
- Branch: main
- Worktree (optional):

## Current Execution Item

- Execution Item ID: none
- Source System: manual-default
- Source Ref: docs/notes-kernel-execution-backlog.md
- Title: No active long-run row
- Status: blocked
- Why This Item Now: `.bagakit/long-run/next-action.json` still has no actionable item. `K-020: adapter projection metrics and denial surfaces` has now been completed through the feat-task harness, so long-run should remain explicitly idle until the next manual row is defined.

## Acceptance Criteria

- [ ] Refresh long-run rows before using `ralphloop` again.
- [ ] Keep handoff aligned with `next-action.json` while the current implementation wave runs through feat-task harness instead.

## Execution Plan

1. Keep long-run explicitly idle until the next row is deliberately created.
2. When this slice is complete, either reopen long-run with a fresh manual row or keep handoff explicitly idle.
3. Do not point handoff at stale completed rows.

## Files To Touch

- `.bagakit/long-run/bk-execution-table.json`
- `.bagakit/long-run/next-action.json`
- `.bagakit/long-run/bk-execution-handoff.md`
- `docs/notes-kernel-execution-backlog.md`

## Commands To Run

```bash
bash .bagakit/long-run/check_and_resume.sh
```

## Expected Verification

- Gate / verification command: `bash .bagakit/long-run/check_and_resume.sh`
- Expected result: `next-action.json` and handoff remain aligned; there is no stale active row while current implementation work proceeds through feat-task harness.

## Results

- Summary: The long-run queue is currently idle. `K-020: adapter projection metrics and denial surfaces` has landed through the feat harness, and there is still no new actionable row.
- Tests: `go test ./...`, `make lint`, and `make check` are green after the `K-020` implementation wave.
- Gate / Verification: `.bagakit/long-run/next-action.json` remains `next_row: null`, which now matches this handoff.

## Response Driver Snapshot

```text
[[BAGAKIT]]
- LivingDoc: long-run handoff refreshed so it no longer points at a completed row while the next adapter-side slice is executed through the feat-task harness.
- LongRun: Item=none; Status=blocked; Confidence=0.96; Evidence=next_row null | K-020 closed | repository gates green; Next=bash .bagakit/long-run/check_and_resume.sh
```

## Risks / Open Questions

- Risks: The main process risk is letting handoff drift from `next-action.json` again while work happens outside the long-run loop.
- Rollback: If a new long-run phase is not ready, keep handoff explicitly idle instead of pointing it at a guessed next row.
- Unblock Action (if blocked): Re-run `bash .bagakit/long-run/check_and_resume.sh`; if a new row is created later, let the generated files advance together.

## Next Run

- Primary: `bash .bagakit/long-run/check_and_resume.sh`
- Fallback: `bash .bagakit/long-run/ralphloop.sh pulse --endless`
- Resume command: `bash .bagakit/long-run/check_and_resume.sh`

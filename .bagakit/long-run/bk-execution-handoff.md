# BK Execution Handoff

## Run Metadata

- Updated At (UTC): 2026-04-04T04:05:00Z
- Updated By: codex
- Branch: main
- Worktree (optional):

## Current Execution Item

- Execution Item ID: none
- Source System: manual-default
- Source Ref: docs/notes-kernel-execution-backlog.md
- Title: No active long-run row
- Status: blocked
- Why This Item Now: `.bagakit/long-run/next-action.json` still has no actionable row. `K-027: external benchmark mapping / evaluation feasibility` is now complete, and the recommended next wave is `K-028: lightweight Terminal-Bench comparison prototype`. Long-run should remain explicitly idle until that next manual row is deliberately created.

## Acceptance Criteria

- [ ] Refresh long-run rows before using `ralphloop` again.
- [ ] Keep handoff aligned with `next-action.json` while the current benchmark-mapping slice is fully closed.

## Execution Plan

1. Keep long-run explicitly idle until the next row is deliberately created.
2. When the next slice is selected, reopen handoff with that new row instead of pointing at the closed benchmark-mapping feat.
3. Do not let handoff drift from `next-action.json` while work proceeds through feat-task harness.

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
- Expected result: `next-action.json` and handoff remain aligned; there is no stale active row while the repository is back in a clean idle state.

## Results

- Summary: `K-027: external benchmark mapping / evaluation feasibility` is complete. The repository now has a checked-in native scenario inventory, Terminal-Bench and SWE-bench-Live mapping artifacts, and guardrail tests; the recommended next wave is `K-028: lightweight Terminal-Bench comparison prototype`, but no new row is active yet.
- Tests: `go test ./benchmarks/external_mapping ./benchmarks/simsh_native_reference -count=1`, `go test ./...`, `make lint`, and `make check`.
- Gate / Verification: `.bagakit/long-run/next-action.json` remains `next_row: null`, and this handoff again matches that idle state.

## Response Driver Snapshot

```text
[[BAGAKIT]]
- LivingDoc: backlog and handoff refreshed so `K-027` is closed and the repo is explicitly idle with `K-028` recommended as the next wave.
- LongRun: Item=none; Status=blocked; Confidence=0.97; Evidence=next_row null | K-027 mapping artifacts landed | K-028 recommended but not started; Next=bash .bagakit/long-run/check_and_resume.sh
```

## Risks / Open Questions

- Risks: The main process risk is letting handoff drift from the now-idle `next-action.json` state again while future work happens outside the long-run loop.
- Rollback: If the next long-run phase is not ready, keep handoff explicitly idle instead of pointing it at a guessed future slice.
- Unblock Action (if blocked): Re-run `bash .bagakit/long-run/check_and_resume.sh`; if a future row is created, let the generated files advance together.

## Next Run

- Primary: `bash .bagakit/long-run/check_and_resume.sh`
- Fallback: `bash .bagakit/long-run/ralphloop.sh pulse --endless`
- Resume command: `bash .bagakit/long-run/check_and_resume.sh`

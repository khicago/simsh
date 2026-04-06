# Feat Proposal: f-20260406-v0-3-1-patch-release-truth-cleanup

## Why
- `v0.3.0` is already tagged, but several release-facing docs still describe it as upcoming.
- Before cutting `v0.3.1`, the repository should stop mixing “historical v0.3.0 closeout” language with “current patch candidate” language.

## Goal
- Align release-facing docs and process state with the fact that v0.3.0 is already tagged and current main is the candidate patch line for v0.3.1.

## Scope
- In scope:
  - README release status wording
  - current release-readiness and migration docs
  - backlog/handoff wording for the next deliberate action
  - one explicit patch-release note or readiness artifact for v0.3.1
- Out of scope:
  - new runtime features
  - benchmark changes beyond already-refreshed evidence
  - moving or retagging `v0.3.0`

## Impact
- Code paths:
  - `README.md`
  - `docs/notes-v0-3-0-release-readiness.md`
  - `docs/notes-v0-2-x-to-v0-3-0-migration.md`
  - `docs/notes-kernel-execution-backlog.md`
  - `docs/must-guidebook.md`
  - `docs/must-sop.md`
  - `.bagakit/long-run/bk-execution-handoff.md`
- Tests:
  - `go test ./...`
  - `make check`
- Rollout notes:
  - Keep the slice docs/process-only. This is patch-release truth cleanup, not a new engineering wave.

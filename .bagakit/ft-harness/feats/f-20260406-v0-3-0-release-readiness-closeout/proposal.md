# Feat Proposal: f-20260406-v0-3-0-release-readiness-closeout

## Why
- `K-031` closed the last major technical release-gate ambiguity around high-latency mounts.
- The remaining work is release-facing alignment: docs, migration guidance, benchmark evidence freshness, and one explicit closeout checklist for the `v0.3.0` line.

## Goal
- Align release-facing docs, migration guidance, and current benchmark evidence so the repository can explicitly justify a v0.3.0 release cut without introducing a new feature wave.

## Scope
- In scope:
  - refresh the checked-in benchmark evidence required by the release story
  - add one explicit release-readiness checklist/closeout doc for `v0.3.0`
  - align README, migration docs, backlog, and handoff with the actual current state
- Out of scope:
  - new runtime primitives
  - new benchmark families
  - version-tagging or publishing automation

## Impact
- Code paths:
  - `benchmarks/simsh_native_reference/reports/*`
  - `benchmarks/terminal_bench_compare/reports/*`
  - `benchmarks/paired_uplift/reports/*`
  - `docs/notes-v0-2-x-to-v0-3-0-migration.md`
  - `docs/notes-kernel-execution-backlog.md`
  - `README.md`
  - `docs/must-guidebook.md`
  - `.bagakit/long-run/bk-execution-handoff.md`
- Tests:
  - `make benchmark-refresh`
  - `make benchmark-uplift`
  - `go test ./...`
  - `make check`
- Rollout notes:
  - Keep the slice release-readiness-only: align and evidence the current line instead of sneaking in another feature wave.

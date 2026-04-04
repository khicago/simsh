# Feat Proposal: f-20260404-automate-terminal-bench-comparison-freshness

## Why
- `K-028` proved the Terminal-Bench comparison prototype is valuable, but it still depends on manual regeneration of the native baseline plus the checked-in comparison artifact/report pair.
- The next missing layer is freshness discipline, not broader benchmark scope.

## Goal
- Add one deterministic refresh path that regenerates the checked-in native benchmark baseline and the Terminal-Bench comparison artifact/report pair without broadening the benchmark scope or creating a second harness.

## Scope
- In scope:
  - One explicit refresh entrypoint for the native baseline plus Terminal-Bench comparison artifact/report pair.
  - One narrow guard that fails if the checked-in comparison pair drifts from the refresh path.
  - Docs/research/memory updates that codify the refresh path and its boundaries.
- Out of scope:
  - Broadening the prototype beyond one direct-fit and one translated slice.
  - Adopting Terminal-Bench wholesale.
  - Adding a second external benchmark family or a new benchmark framework.

## Impact
- Code paths:
  - `benchmarks/external_mapping/*`
  - `benchmarks/simsh_native_reference/reports/*`
  - `benchmarks/terminal_bench_compare/*`
  - `Makefile`
  - `task_outputs/research/*`
  - `docs/notes-kernel-execution-backlog.md`
  - `.bagakit/long-run/bk-execution-handoff.md`
- Tests:
  - `go test ./benchmarks/... -count=1`
  - `go test ./...`
  - `make lint`
  - `make check`
- Rollout notes:
  - Keep refresh automation downstream from the native benchmark and prototype scope SSOT.
  - Do not turn the refresh command into a second benchmark harness.

# Feat Proposal: f-20260403-lightweight-terminal-bench-comparison-prototype

## Why
- `K-027` proved that Terminal-Bench is the only near-term external family with a meaningful direct fit.
- The next missing layer is not more mapping, but one concrete downstream comparison/export path that demonstrates how a direct-fit slice and a translated slice can be compared without weakening the native benchmark SSOT.

## Goal
- Build a small downstream Terminal-Bench comparison/export layer around one direct-fit native scenario and one translated scenario without changing native benchmark semantics or broadening simsh scope.

## Scope
- In scope:
  - A lightweight Terminal-Bench comparison builder that consumes native benchmark report data plus checked-in K-027 mapping/config artifacts.
  - One direct-fit comparison item centered on `inspect_edit_write_loop`.
  - One translated comparison item centered on either `relative_navigation_session` or `cancel_timeout_interruptions`.
  - Narrow tests and one compact report artifact that prove the comparison layer works.
- Out of scope:
  - Full Terminal-Bench adoption.
  - Changes to runtime semantics or native benchmark scenario definitions.
  - Comparison support for all translated/excluded scenarios.

## Impact
- Code paths:
  - `benchmarks/internal/*`
  - `benchmarks/simsh_native_reference/*`
  - `benchmarks/external_mapping/*`
  - `benchmarks/terminal_bench_compare/*`
  - `task_outputs/research/*`
  - `docs/notes-kernel-execution-backlog.md`
  - `.bagakit/long-run/bk-execution-handoff.md`
- Tests:
  - `go test ./benchmarks/... -count=1`
  - `go test ./...`
- Rollout notes:
  - Keep the comparison layer downstream from the native benchmark and mapping SSOT.
  - Treat translated comparison as proof of method, not as broad benchmark coverage.

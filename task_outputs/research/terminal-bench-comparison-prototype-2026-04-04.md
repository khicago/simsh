# Terminal-Bench Comparison Prototype

## Scope

- External family: `Terminal-Bench`
- Scope SSOT: `benchmarks/terminal_bench_compare/prototype_scope.json`
- Native input baseline: `benchmarks/simsh_native_reference/reports/baseline-20260404.json`
- Machine artifact: `benchmarks/terminal_bench_compare/reports/prototype-baseline-20260404.json`
- Human summary: `benchmarks/terminal_bench_compare/reports/prototype-baseline-20260404.md`

## Selected slices

- Direct fit: `inspect_edit_write_loop`
- Translated proof: `relative_navigation_session`

## Why this cut

- `inspect_edit_write_loop` is the cleanest existing `as_is` fit in `terminal_bench_mapping.json`.
- `relative_navigation_session` is the smallest translated slice that pressures terminal-task navigation without dragging in broader adapter or dynamic-workload semantics.
- The prototype stays downstream from the native benchmark SSOT and does not introduce benchmark-only scenario ids or a second scenario catalog.

## Explicit non-goals

- Do not adopt Terminal-Bench wholesale.
- Do not widen beyond one translated proof slice.
- Do not mutate native benchmark semantics to look more benchmark-compatible.

## Freshness note

- The native baseline is now treated as a refreshed snapshot input, not a byte-stable golden artifact.
- The byte-guarded pair is the checked-in Terminal-Bench comparison JSON/Markdown output derived from the current baseline, current mapping, and current prototype scope.

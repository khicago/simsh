---
title: Use prepared ops to remove per-exec normalize overhead
kind: decision
status: inbox
tags:
  - decision
  - performance
  - engine
sources:
  - pkg/engine/orchestrator.go
  - pkg/engine/runtime/runtime_stack.go
  - pkg/engine/engine_benchmark_test.go
  - docs/refs/notes-execute-preflight-performance-refs.md
created: 2026-03-02T17:32:41Z
---

## Candidate
- Problem: the hot path was not command execution itself, but per-exec preparation (`normalizeOps` + mount wrapping + alias merge), causing high alloc/op and GC pressure.
- Decision:
  - Introduce `PreparedOps` (`Engine.PrepareOps` + `Engine.ExecutePrepared`) so ops normalization happens once and execution reuses compiled callbacks.
  - Make `runtime.Stack` prepare once in `New` and always execute through `ExecutePrepared`.
  - Keep `Engine.Execute` behavior-compatible by delegating to prepare+execute internally.
  - Simplify alias expansion hot path to avoid re-normalizing alias maps on every command dispatch.
- Validation:
  - Functional: `go test ./...` passes.
  - Performance benchmarks (5 runs): prepared path reduces both latency and allocations.
    - `echo`: ~1.8us / 3336B / 56 allocs -> ~0.8-0.96us / 1048B / 22 allocs.
    - `echo > file`: ~2.35-2.55us / 3536B / 61 allocs -> ~1.34-1.43us / 1248B / 27 allocs.

## Promote To
- `docs/.bagakit/memory/decision-execute-preflight-prepared-ops.md` (curated), or
- `docs/<type>-<topic>.md` (normative/deep guide)

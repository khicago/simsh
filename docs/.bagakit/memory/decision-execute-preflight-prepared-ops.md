---
title: Use prepared ops to remove per-exec normalize overhead
kind: decision
confidence: high
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
updated: 2026-03-23T00:00:00Z
---

## Context
Execution hot-path profiling showed that a large part of the cost came from per-exec setup, not from command execution itself. The repeated work was mainly ops normalization, mount wrapping, and alias merge.

## Decision
- Introduce `PreparedOps` through `Engine.PrepareOps` and `Engine.ExecutePrepared` so normalization happens once and execution reuses compiled callbacks.
- Make `runtime.Stack` prepare once during construction and execute through the prepared path by default.
- Keep `Engine.Execute` behavior-compatible by delegating internally to prepare plus execute.
- Keep alias expansion on the hot path cheap enough that repeated command execution does not re-normalize the same structures every time.

## Rationale
- The runtime should spend cycles on execution and policy enforcement, not on rebuilding equivalent wrappers for every command.
- This optimization keeps the external contract stable while improving the common path for both one-shot engine use and session-backed runtime use.
- Prepared execution is especially important once the runtime becomes a reusable kernel inside a harness, because repeated small commands dominate many agent workflows.

## Validation
- `go test ./...` passed when this change landed.
- Benchmarks showed clear reductions in both latency and allocation pressure.
- The main observed win was that prepared paths removed repeated setup overhead from tiny commands and redirection-heavy commands alike.

## Scope
- Applies when touching engine execution hot paths or considering regressions that reintroduce per-exec normalization.
- Does not imply that every future optimization should become a cache; the rule is to remove repeated deterministic preparation work first.

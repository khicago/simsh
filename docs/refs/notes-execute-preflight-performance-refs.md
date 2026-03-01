---
title: Execute Preflight Performance References
required: false
sop:
  - Use this list before major runtime-performance refactors around per-exec setup overhead.
  - Update the actionable-takeaways section when adopting a new optimization strategy.
  - Regenerate docs/must-sop.md after SOP/frontmatter changes.
---

# Execute Preflight Performance References

## Context
This reference list targets one class of latency issue: **command execution is fast, but per-exec preparation is expensive** (normalization, wrapper construction, alias/env bootstrap, and short-lived allocations).

## Curated Sources (Articles + Papers)

1. Go diagnostics workflow (official)
- Link: https://go.dev/doc/diagnostics
- Why it matters: establishes the standard measure-first flow (profiling, tracing, benchmarking) before changing code.

2. Profiling Go Programs (Go blog)
- Link: https://go.dev/blog/profiling-go-programs
- Why it matters: shows how to identify true hot paths and GC pressure with pprof instead of inferring from wall-clock only.

3. A Guide to the Go Garbage Collector (official)
- Link: https://go.dev/doc/gc-guide
- Why it matters: explains allocation-rate vs GC-cost trade-offs; directly relevant when per-exec prep creates many short-lived objects.

4. Profile-guided optimization (official)
- Link: https://go.dev/doc/pgo
- Why it matters: provides production-profile-driven optimization to improve hot code paths without manual micro-tuning everywhere.

5. `golang.org/x/perf` + `benchstat` workflow
- Link: https://pkg.go.dev/golang.org/x/perf
- Why it matters: offers statistically sound benchmark comparisons for before/after optimization validation.

6. `sync.Map` design intent (official package docs)
- Link: https://pkg.go.dev/sync#Map
- Why it matters: highlights read-heavy optimization patterns and safe trade-offs for cache-like access paths.

7. Read-Copy Update (RCU) memory/latency trade-offs (paper)
- Link: https://arxiv.org/abs/2406.11607
- Why it matters: formalizes the read-mostly design principle (pay update cost once, keep read path cheap), matching precompute-once execute-many flows.

8. A Verification Perspective on RCU (paper)
- Link: https://arxiv.org/abs/1512.05438
- Why it matters: gives a correctness model for read-mostly concurrency patterns, useful when introducing cached/prepared execution artifacts.

## Actionable Takeaways for simsh

- Separate cold path from hot path:
  - Cold path: ops validation, mount routing composition, alias/env/rc normalization.
  - Hot path: parse + dispatch + builtin/external execution.

- Prefer precompute/reuse for read-mostly execution:
  - Build a prepared execution snapshot once per runtime instance.
  - Reuse it across commands instead of re-running normalization every `Execute`.

- Validate optimizations with statistical rigor:
  - Keep microbenchmarks for representative commands (e.g. `echo`, redirect write).
  - Compare with `benchstat` and track allocs/op, B/op, and ns/op.

- Keep correctness guardrails:
  - Add behavior-equivalence tests between default execute path and prepared execute path.
  - Keep policy/timeout and path-access semantics unchanged.

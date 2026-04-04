---
title: Terminal-Bench refresh keeps the native baseline as a fresh snapshot, not a byte-stable golden file
kind: decision
tags:
  - decision
  - benchmark
  - evaluation
  - freshness
sources:
  - benchmarks/refresh_terminal_bench_compare/main.go
  - benchmarks/external_mapping/refresh.go
  - benchmarks/simsh_native_reference/reports/baseline-20260404.json
  - benchmarks/terminal_bench_compare/reports/prototype-baseline-20260404.json
  - task_outputs/research/terminal-bench-comparison-freshness-automation-2026-04-04.md
created: 2026-04-04T07:20:00Z
confidence: high
updated: 2026-04-04T07:20:00Z
---

## Context
- `K-029` adds a deterministic refresh path for the checked-in native baseline and the checked-in Terminal-Bench comparison pair.
- The native baseline contains volatile fields such as `generated_at` and duration-derived latencies.

## Decision
- Treat the native baseline as a freshness snapshot that is expected to change when refreshed.
- Treat the comparison JSON/Markdown pair as the byte-guarded downstream artifact pair.
- Keep refresh automation orchestration-only: it reruns the current producers, but it does not redefine benchmark scope or mapping semantics.

## Why
- This preserves a useful distinction between:
  - fresh measured baseline inputs
  - stable derived comparison outputs
- It avoids brittle tests that pretend timing fields are deterministic.
- It keeps the comparison layer downstream from the native benchmark rather than trying to freeze the native benchmark into a golden artifact.

---
title: External benchmark mapping keeps native benchmark identity as SSOT
kind: decision
tags:
  - decision
  - benchmark
  - evaluation
  - planning
sources:
  - benchmarks/internal/scenarios/catalog.go
  - benchmarks/external_mapping/scenario_inventory.json
  - benchmarks/external_mapping/terminal_bench_mapping.json
  - benchmarks/external_mapping/swe_bench_live_mapping.json
  - task_outputs/research/external-benchmark-mapping-feasibility-2026-04-04.md
  - docs/notes-kernel-execution-backlog.md
created: 2026-04-04T03:40:00Z
confidence: high
updated: 2026-04-04T03:40:00Z
---

## Context
- `K-027` added a checked-in external benchmark mapping layer on top of the existing native benchmark suite.
- The design risk was letting benchmark-fit work mutate the native suite or overclaim that curated mapping metadata is runtime truth.

## Decision
- Stable native benchmark scenario ids and categories remain the primary SSOT.
- External benchmark mapping stays downstream as an evaluation layer.
- Task-shape summaries and truth-surface lists are curated evaluation metadata, not benchmark execution contract fields.
- Mapping should classify scenarios as `as_is`, `translated`, or `excluded` rather than stretching external benchmark families to fit every native proof slice.

## Why
- This keeps evaluation work from rewriting the native benchmark around external tastes.
- It preserves the strongest current project asset: explicit proof layering around runtime truth.
- It lets the repo answer benchmark-fit questions deterministically without pretending current scope already matches a larger benchmark suite.

## Scope
- Applies to external benchmark mapping work such as Terminal-Bench or SWE-bench-Live fit studies.
- Does not prevent future comparison layers; it constrains them to remain downstream from native benchmark identity.

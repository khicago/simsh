---
title: Terminal-Bench prototype stays at one direct-fit slice plus one translated proof slice
kind: decision
tags:
  - decision
  - benchmark
  - evaluation
  - planning
sources:
  - benchmarks/terminal_bench_compare/prototype_scope.json
  - benchmarks/terminal_bench_compare/reports/prototype-baseline-20260404.json
  - task_outputs/research/terminal-bench-comparison-prototype-2026-04-04.md
  - docs/notes-kernel-execution-backlog.md
created: 2026-04-04T04:15:00Z
confidence: high
updated: 2026-04-04T04:15:00Z
---

## Context
- `K-028` adds the first checked-in external comparison prototype on top of the native benchmark suite and K-027 mapping artifacts.
- The main risk was letting the first comparison layer expand into a second benchmark suite or a hidden feature wave.

## Decision
- Keep the Terminal-Bench comparison prototype at exactly two slices:
  - one `as_is` direct-fit slice: `inspect_edit_write_loop`
  - one `translated` proof slice: `relative_navigation_session`
- Keep the prototype downstream from:
  - native benchmark ids/categories
  - checked-in native baseline report
  - checked-in Terminal-Bench mapping artifacts
- Do not let the comparison layer introduce benchmark-only scenario ids or mutate the native benchmark suite.

## Why
- This proves both comparison modes that matter now:
  - direct fit
  - narrow translation
- It gives a concrete external comparison artifact without broadening the evaluation surface prematurely.
- It preserves the project’s strongest current asset: native proof layering and explicit truth surfaces.

## Scope
- Applies to the lightweight Terminal-Bench comparison prototype and its refresh/expansion decisions.
- Does not prevent future slices, but future additions should be separate feats rather than quiet scope creep inside the prototype.

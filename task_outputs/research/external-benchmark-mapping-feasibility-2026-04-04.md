---
title: External Benchmark Mapping Feasibility
summary: Result-oriented writeup for K-027, capturing how the current simsh native benchmark maps to Terminal-Bench and SWE-bench-Live, what remains out of scope, and the recommended next build wave.
date: 2026-04-04
tags:
  - research
  - benchmark
  - evaluation
  - planning
---

# Goal

`K-027` answers a narrow question:

> given the current native benchmark suite, what can `simsh` compare to external benchmark families as-is, what needs translation, and what should remain out of scope?

This is an evaluation-feasibility slice, not benchmark adoption.

# Artifacts

Checked-in artifacts:

- `benchmarks/internal/scenarios/catalog.go`
- `benchmarks/external_mapping/scenario_inventory.json`
- `benchmarks/external_mapping/terminal_bench_mapping.json`
- `benchmarks/external_mapping/swe_bench_live_mapping.json`
- `benchmarks/external_mapping/mapping_guard_test.go`

# Result

## Terminal-Bench

- `as_is`: 1
- `translated`: 4
- `excluded`: 4

Interpretation:
- Terminal-Bench is still the strongest near-term external family.
- The clearest direct fit is `inspect_edit_write_loop`.
- Relative navigation, mount-boundary pressure, adapter-backed lifecycle, and interruption pressure are useful only with a narrow translation layer.
- Command-namespace validation, trace-consumable planning, second-adapter seam proof, and composition/evolution stress should stay outside direct Terminal-Bench mapping.

## SWE-bench-Live

- `as_is`: 0
- `translated`: 4
- `excluded`: 5

Interpretation:
- SWE-bench-Live remains the dynamic-workload reference, not the first direct integration target.
- Its strongest fit is as a pressure source for evolving-task and review-loop thinking.
- It is not the right next wave for direct implementation because the current native suite does not match it closely enough without broad translation.

# Design Conclusion

The mapping work confirms three things:

1. Native benchmark scenario ids and categories should stay canonical.
2. Task-shape summaries and truth-surface lists are valuable, but they are curated evaluation metadata rather than runtime contract truth.
3. The next wave should be a very small Terminal-Bench comparison prototype, not a larger benchmark adoption wave.

# Recommended Next Wave

## K-028

**Prototype a lightweight Terminal-Bench comparison layer**

Why:
- it builds on the one scenario that already fits directly;
- it pressures the translation boundary without forcing a larger benchmark harness;
- it keeps `simsh` evaluation-forward without widening the product/runtime scope.

Suggested first slice:
- direct comparison around `inspect_edit_write_loop`
- optionally one translated scenario, likely either `relative_navigation_session` or `cancel_timeout_interruptions`

# Non-Goals

- Do not weaken the native benchmark suite.
- Do not introduce full Terminal-Bench infrastructure.
- Do not turn this into full environment synthesis.
- Do not treat SWE-bench-Live as an implementation target yet.

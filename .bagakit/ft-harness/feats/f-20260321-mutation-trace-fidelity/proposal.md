# Feat Proposal: f-20260321-mutation-trace-fidelity

## Why
- `ExecutionTrace` is already directionally useful, but mutation-heavy operations still have fidelity gaps that reduce planner and reviewer trust.
- If traces undercount or misclassify important mutations, the kernel cannot yet claim that `ExecutionResult` and `ExecutionTrace` are full SSOT for machine consumption.

## Goal
- Make ExecutionTrace and related resource summaries faithful enough for planners and reviewers to trust mutation-heavy operations without hidden heuristics.

## Scope
- In scope:
- auditing and improving bytes-read / bytes-written accounting for core file mutations
- verifying mutation and denial coverage for write, append, edit, and remove flows
- tightening trace truth at core file-operation seams
- Out of scope:
- unrelated filesystem capability changes except where needed to make trace truthful
- product-specific trace consumers
- broader causality graph design beyond the current kernel need

## Impact
- Code paths:
- `pkg/engine/trace_collector.go`
- `pkg/engine/execution_result_test.go`
- core mutation implementations as needed for accurate accounting
- Tests:
- mutation trace tests for write/append/edit/remove
- deny-path and resource-summary assertions
- Rollout notes:
- keep trace directly planner-friendly; avoid drifting into audit-log duplication

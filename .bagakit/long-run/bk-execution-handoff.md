# BK Execution Handoff

## Run Metadata

- Updated At (UTC): 2026-03-31T16:28:00Z
- Updated By: codex
- Branch: main
- Worktree (optional):

## Current Execution Item

- Execution Item ID: EXEC::manual-20260331-reference-workflow-transition-control-plane
- Source System: manual-default
- Source Ref: pkg/adapter/reference/adapter.go
- Title: Implement an explicit workflow-transition control plane over managed `/memory` views
- Status: todo
- Why This Item Now: The freshness lifecycle, evidence review, and contract-refine slices are complete. The next current-phase gap is no longer freshness truth; it is workflow-transition truth. The reference adapter still derives workflow state mostly from trace heuristics, so the next implementation slice is to add an explicit adapter-local transition seam without turning `/memory` into a writable backdoor.

## Acceptance Criteria

- [ ] The reference adapter exposes a minimal workflow-transition control plane that can record explicit status or reason updates while preserving the existing trace-derived defaults.
- [ ] `/memory/workflows.json`, `/memory/workflows.md`, and `/memory/status.json` surface both the current workflow status and whether it came from trace evidence or explicit adapter control-plane action.
- [ ] Adapter tests cover checkpoint and resume with both trace-driven and control-plane-driven workflow transitions.
- [ ] `go test ./pkg/adapter/reference ./pkg/engine/runtime` exits 0.

## Execution Plan

1. Add the smallest explicit workflow-transition control-plane seam that stays adapter-local and auditable.
2. Preserve trace-derived defaults, but let explicit control-plane transitions annotate or override them when the adapter owns that policy.
3. Surface transition provenance in managed `/memory` views so a harness can tell whether state came from trace evidence or explicit control-plane action.
4. Add direct adapter tests before widening the benchmark again.

## Files To Touch

- `pkg/adapter/reference/adapter.go`
- `pkg/adapter/reference/adapter_test.go`
- `pkg/adapter/reference/adapter_helpers_test.go`

## Commands To Run

```bash
go test ./pkg/adapter/reference ./pkg/engine/runtime
go test ./pkg/adapter/reference -run 'TestReferenceAdapter'
```

## Expected Verification

- Gate / verification command: `go test ./pkg/adapter/reference ./pkg/engine/runtime`
- Expected result: workflow transition control-plane behavior is covered directly, persists across checkpoint/resume, and does not regress adapter/runtime tests.

## Results

- Summary: The freshness lifecycle, evidence review, and contract refinement slices are complete. The next actionable row is now workflow-transition control-plane implementation.
- Tests: `go test ./pkg/adapter/reference ./pkg/engine/runtime ./benchmarks/simsh_native_reference` and `go test ./...` are green after the prior slices.
- Gate / Verification: `.bagakit/long-run/next-action.json` now points to `manual-20260331-reference-workflow-transition-control-plane`.

## Response Driver Snapshot

```text
[[BAGAKIT]]
- LivingDoc: current-phase handoff advanced to the workflow-transition implementation row after freshness implementation, review, and contract refinement all closed.
- LongRun: Item=manual-20260331-reference-workflow-transition-control-plane; Status=todo; Confidence=0.89; Evidence=K-012/K-013 closed | next-action advanced | tests green; Next=bash .bagakit/long-run/check_and_resume.sh
```

## Risks / Open Questions

- Risks: The main risk is overfitting the reference adapter to one product workflow model. The transition seam should stay minimal, explicit, and clearly downstream from kernel contracts.
- Rollback: If explicit transitions become too opinionated, keep the provenance fields and narrow the control plane to the smallest generic transition hook that still matches tests.
- Unblock Action (if blocked): Re-run `bash .bagakit/long-run/check_and_resume.sh`; if the row still does not select cleanly, resync generated long-run state before touching code.

## Next Run

- Primary: `bash .bagakit/long-run/ralphloop-runner.sh`
- Fallback: `bash .bagakit/long-run/ralphloop.sh pulse --endless`
- Resume command: `bash .bagakit/long-run/check_and_resume.sh`

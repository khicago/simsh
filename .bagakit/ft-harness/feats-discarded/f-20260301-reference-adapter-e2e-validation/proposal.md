# Feat Proposal: f-20260301-reference-adapter-e2e-validation

## Why
- New kernel and adapter contracts are still provisional until at least one real adapter-backed workload proves the seam works end to end.
- Without a reference implementation, projection, trace consumption, and memory lifecycle rules are likely to drift from how platforms actually use them.

## Goal
- Implement one adapter-backed reference workload that validates session, trace, projection, and memory contracts end to end.

## Scope
- In scope:
- one reference adapter that exercises RPC-to-file projection or equivalent external-system projection
- end-to-end tests for session create/resume/checkpoint/close
- validation that execution traces are consumed by the adapter rather than merely emitted
- Out of scope:
- additional product-specific adapters
- production rollout for a concrete business workload
- broad multi-agent orchestration features

## Impact
- Code paths:
- adapter/reference implementation package(s)
- end-to-end tests
- validation docs or runbook notes if needed
- Tests:
- adapter-backed end-to-end scenarios
- projection freshness/error handling tests
- trace-driven follow-up behavior tests
- Rollout notes:
- use this feat to decide whether new contracts are stable enough to promote from design docs into long-term runtime SSOT

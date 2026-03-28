# Feat Proposal: f-20260403-adapter-composition-evolution-stress-validation

## Why
- `K-017` through `K-024` proved many adapter truths in isolation: selection, control plane, audit, metrics, second-shape seam validation, shared conformance helpers, and helper self-tests.
- The remaining risk is composition drift: these truth surfaces may each be correct alone but still diverge under a multi-step evolution workload.

## Goal
- Prove that projection, control-plane mutation, freshness/materialization, audit, metrics, and checkpoint/resume stay coherent together under a multi-step adapter evolution workload.

## Scope
- In scope:
  - A reference-adapter stress workload that composes projection reads, control-plane mutation, freshness/materialization changes, denials, and checkpoint/resume.
  - Structural assertions that `/memory/status.json`, `/memory/projections.json`, `/memory/projection_metrics.json`, `/memory/skills_audit.json`, and `/memory/denials.json` stay aligned under that sequence.
  - Backlog and contract docs that record composition/evolution validation as the next proof layer after helper conformance.
- Out of scope:
  - New adapter features or new product domains.
  - A third adapter shape.
  - Registry, marketplace, or remote sync behavior.

## Impact
- Code paths:
  - `pkg/adapter/reference/*_test.go`
  - `benchmarks/simsh_native_reference/*`
- Tests:
  - adapter composition stress tests
  - benchmark stress scenario
- Rollout notes:
  - Prefer one hard composition workload over many shallow scenarios; keep the slice proof-oriented, not feature-oriented.

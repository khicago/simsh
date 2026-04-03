# Feat Proposal: f-20260403-external-benchmark-mapping-evaluation-feasibility

## Why
- `K-026` closed the strategy question: the next highest-value wave is external benchmark mapping, not more runtime nouns.
- `simsh` now needs a checked-in, machine-readable answer for which native benchmark scenarios align with external benchmark families, which require translation, and which should stay out of scope.

## Goal
- Map simsh native benchmark scenarios to external benchmark families, define as-is/translated/excluded boundaries, and add guardrails so mapping stays aligned with native benchmark evolution.

## Scope
- In scope:
  - Canonical native scenario inventory derived from the existing native benchmark suite.
  - Mapping artifacts for Terminal-Bench and SWE-bench-Live.
  - Guardrail tests that fail when native benchmark scenarios drift without mapping updates.
  - Narrow docs/research updates that explain the mapping layer and its boundaries.
- Out of scope:
  - Full external benchmark adoption.
  - New runtime primitives, new adapters, or environment synthesis.
  - Weakening or replacing the native benchmark suite.

## Impact
- Code paths:
  - `benchmarks/simsh_native_reference/*`
  - `benchmarks/external_mapping/*`
  - `benchmarks/internal/scenarios/*`
  - `task_outputs/research/*`
  - `docs/notes-kernel-execution-backlog.md`
  - `docs/architecture-platform-adapter-contract.md`
  - `docs/.bagakit/memory/*`
- Tests:
  - `go test ./benchmarks/external_mapping ./benchmarks/simsh_native_reference -count=1`
  - `go test ./...`
- Rollout notes:
  - Keep the external mapping layer machine-readable and downstream from the native benchmark SSOT.
  - Preserve native benchmark primacy; this feat adds evaluation clarity, not benchmark adoption.

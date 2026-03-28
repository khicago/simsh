# Feat Proposal: f-20260323-realistic-adapter-projections-managed-memory-views

## Why
- The adapter seam is now validated, but the reference adapter still behaves like a toy projection: one mirrored document tree and one flat observations log.
- A more realistic non-core layer should project multiple namespaces, maintain managed `/memory` views, and expose workflow state derived from trace consumption.
- This feat upgrades the reference adapter without pushing product semantics back into core packages.

## Goal
- Upgrade the reference adapter to project resources, maintain richer managed /memory views, and expose adapter-backed workflow state without pushing product semantics into core.

## Scope
- In scope:
  - `/resources` projection alongside `/knowledge_base/reference`
  - richer `/memory` files such as summaries and workflow views
  - adapter-managed workflow state derived from read/write/denial trace evidence
  - benchmark and adapter tests that exercise the richer projection model
- Out of scope:
  - writable mounts for `/memory` or `/resources`
  - product-specific retrieval, ranking, or curation control planes
  - new core-runtime contracts

## Impact
- Code paths:
  - `pkg/adapter/reference/adapter.go`
  - `pkg/adapter/reference/adapter_test.go`
  - `benchmarks/simsh_native_reference/`
- Tests:
  - `go test ./pkg/adapter/reference ./benchmarks/simsh_native_reference ./pkg/engine/runtime`
- Rollout notes:
  - keep the projection deterministic and read-only; richer behavior should appear as projected files, not hidden side effects

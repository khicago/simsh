# Feat Proposal: f-20260323-adapter-projection-metadata-control-plane

## Why
- The reference adapter now projects richer content, but projected objects still lack first-class source/freshness metadata.
- A realistic adapter also needs an explicit control-plane seam for non-core updates, rather than overloading raw content mutation helpers as the only API.
- This feat adds those capabilities without widening any core-runtime contracts.

## Goal
- Add explicit source/freshness metadata and a minimal control-plane API to the reference adapter without widening core contracts.

## Scope
- In scope:
  - projection metadata for documents and resources
  - read-only metadata sidecars/indexes in projected trees and `/memory`
  - explicit adapter methods for upserting projected items and workflow definitions
- Out of scope:
  - core contract/schema changes
  - writable `/memory` or `/resources`
  - product-specific ranking or retrieval logic

## Impact
- Code paths:
  - `pkg/adapter/reference/adapter.go`
  - `pkg/adapter/reference/adapter_test.go`
  - `benchmarks/simsh_native_reference/suite.go`
  - `docs/notes-kernel-execution-backlog.md`
- Tests:
  - `go test ./pkg/adapter/reference ./benchmarks/simsh_native_reference ./pkg/engine/runtime`
- Rollout notes:
  - expose metadata as deterministic projected files, not hidden adapter-only state

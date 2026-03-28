# Feat Proposal: f-20260323-adapter-backed-workload-validation

## Why
- The adapter boundary is documented and lightly covered, but promotion to a stable seam should depend on a committed adapter-backed workload, not only unit tests.
- The benchmark/reference suite currently emphasizes kernel behaviors more than projection lifecycle behaviors.
- This feat exists to make adapter-backed projection and managed-memory lifecycle validation first-class evidence.

## Goal
- Validate the adapter seam with a committed reference workload that exercises projection and managed-memory lifecycle semantics.

## Scope
- In scope:
  - adapter-backed reference workload in committed validation artifacts
  - checks for projection lifecycle, `/memory`, and `/knowledge_base/reference` behavior
- Out of scope:
  - heavy product-layer memory/indexing logic
  - introducing a new adapter framework
  - turning `/memory` into a writable scratch surface

## Impact
- Code paths:
  - `benchmarks/simsh_native_reference/`
  - `pkg/adapter/reference/adapter_test.go`
- Tests:
  - `go test ./pkg/adapter/reference ./benchmarks/simsh_native_reference ./pkg/engine/runtime`
- Rollout notes:
  - build on the existing reference adapter and keep the scenario deterministic

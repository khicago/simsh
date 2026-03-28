# Feat Proposal: f-20260402-skills-control-plane-observability-audit

## Why
- The reference adapter now has a real skill control plane, but it still lacks an explicit audit surface for what changed, why selection moved, and when those changes became session-visible.
- Without a machine-readable audit layer, later adapters or harnesses will have to infer control-plane truth indirectly from projection diffs, which is noisy and brittle.
- This slice deepens the adapter seam without widening core contracts: it makes control-plane behavior observable, not more product-shaped.

## Goal
- Expose machine-readable control-plane audit events and visibility timing for skill add/update/remove without widening core contracts or making /skills writable.

## Scope
- In scope:
  - machine-readable control-plane audit events for skill add/update/remove
  - explicit visibility timing for when a control-plane mutation becomes projection-visible
  - compact human-readable summary views under `/memory`
  - adapter tests and benchmark proof that event audit and projection truth stay aligned
  - canonical docs and backlog updates for the new audit shape
- Out of scope:
  - remote sync or registry workflows
  - writable `/skills` mounts
  - generic event bus infrastructure in core packages
  - broad policy engines beyond the minimal audit surface

## Impact
- Code paths:
  - `pkg/adapter/reference/adapter.go`
  - `pkg/adapter/reference/adapter_helpers_test.go`
  - `pkg/adapter/reference/adapter_test.go`
  - `benchmarks/simsh_native_reference/suite.go`
  - `benchmarks/simsh_native_reference/README.md`
  - `docs/architecture-memory-skills-extension.md`
  - `docs/architecture-platform-adapter-contract.md`
  - `docs/notes-kernel-execution-backlog.md`
  - `.bagakit/long-run/bk-execution-handoff.md`
- Tests:
  - `go test ./pkg/adapter/reference ./benchmarks/simsh_native_reference ./pkg/engine/runtime`
  - `go test ./...`
  - `make lint`
  - `make check`
- Rollout notes:
  - audit truth stays adapter-local and rides alongside projection truth
  - event surfaces should be compact and machine-readable, not prose logs

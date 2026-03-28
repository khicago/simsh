# Feat Proposal: f-20260401-skills-control-plane-lifecycle

## Why
- The reference adapter now has explicit `/skills` selection truth, but skills are still effectively static after adapter construction.
- The architecture already expects a control-plane seam for skill register, update, and unregister flows, and the reference adapter does not yet exercise that responsibility.
- This is the next realistic non-core slice because it deepens the adapter boundary while preserving the current read-only `/skills` projection.

## Goal
- Add a minimal explicit skill control plane to the reference adapter so skill entries can be registered, updated, and removed without widening core contracts or making /skills writable.

## Scope
- In scope:
  - explicit adapter-local APIs for adding, updating, and removing skill entries
  - deterministic re-projection of `/skills`, `/skills/_index.json`, and `/memory/projections.json` after control-plane changes
  - adapter tests and reference benchmark proof that skill updates change selection truth without making `/skills` writable
  - backlog and architecture updates for the minimal reference control-plane shape
- Out of scope:
  - remote skill sync or marketplace metadata
  - product-style skill registry workflows
  - writable `/skills` mounts
  - moving skill evolution logic into core runtime packages

## Impact
- Code paths:
  - `pkg/adapter/reference/adapter.go`
  - `pkg/adapter/reference/adapter_helpers_test.go`
  - `pkg/adapter/reference/adapter_test.go`
  - `benchmarks/simsh_native_reference/suite.go`
  - `docs/architecture-memory-skills-extension.md`
  - `docs/architecture-platform-adapter-contract.md`
  - `docs/notes-kernel-execution-backlog.md`
  - `.bagakit/long-run/bk-execution-handoff.md`
- Tests:
  - `go test ./pkg/adapter/reference ./benchmarks/simsh_native_reference ./pkg/engine/runtime`
  - `go test ./...`
- Rollout notes:
  - keep `/skills` read-only; only the adapter control plane mutates skill state
  - selection truth must remain derived from the same SSOT after control-plane changes

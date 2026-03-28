# Feat Proposal: f-20260402-adapter-projection-metrics-denial-surfaces

## Why
- The reference adapter now has realistic projection namespaces, explicit skill selection truth, a minimal skill control plane, and machine-readable control-plane audit.
- But Stage D is still only partially complete: there is no compact machine-readable projection metrics surface, and denials still live mostly as raw path lists rather than a dedicated adapter-facing view.
- This slice should add observability that is real, not decorative: metrics and denial surfaces that help harnesses reason about adapter state without inventing fake cache-hit semantics.

## Goal
- Expose machine-readable projection metrics and denial/policy views for the reference adapter without inventing fake cache-hit semantics or widening core contracts.

## Scope
- In scope:
  - machine-readable projection metrics for the reference adapter
  - compact denial or policy surfaces under `/memory`
  - explicit projection-generation and build-latency visibility where it is real and measurable
  - benchmark and adapter tests that prove metrics and denial views stay aligned with projection and control-plane truth
- Out of scope:
  - fake cache-hit metrics when no cache exists
  - remote sync, registry, or marketplace concerns
  - generic policy engines in core packages
  - broad observability platforms beyond the minimal reference adapter seam

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
  - metrics must be truthful and compact
  - cache-oriented fields are deferred until a real cache exists

# Feat Proposal: f-20260403-second-adapter-seam-validation

## Why
- The current adapter seam is now well exercised, but almost entirely through one increasingly capable reference adapter.
- That is valuable, but it risks overfitting the seam to one business shape.
- The next high-value validation step is a second adapter with a deliberately smaller surface so we can prove which parts of the seam are truly generic and which are just conveniences of the current reference implementation.

## Goal
- Add a second, smaller adapter shape that projects a resource-first workspace and prove the adapter seam does not overfit the current reference adapter.

## Scope
- In scope:
  - add a second adapter package with a smaller, resource-first shape
  - prove that a valid adapter can omit `/skills`, curation, workflow control-plane, and still fit the same lifecycle/projection seam
  - add focused adapter tests and at least one benchmark scenario that uses the second adapter
  - update canonical docs/backlog to describe the new seam evidence
- Out of scope:
  - enriching the existing `reference` adapter further
  - introducing product registry, retrieval, or orchestration semantics
  - widening core contracts to accommodate one adapter’s convenience
  - building a full second product domain

## Impact
- Code paths:
  - `pkg/adapter/<new-adapter>/`
  - `pkg/adapter/<new-adapter>/*_test.go`
  - `benchmarks/simsh_native_reference/suite.go`
  - `benchmarks/simsh_native_reference/README.md`
  - `docs/architecture-platform-adapter-contract.md`
  - `docs/notes-kernel-execution-backlog.md`
  - `.bagakit/long-run/bk-execution-handoff.md`
- Tests:
  - `go test ./pkg/adapter/... ./benchmarks/simsh_native_reference ./pkg/engine/runtime`
  - `go test ./...`
  - `make lint`
  - `make check`
- Rollout notes:
  - the second adapter should be smaller than `reference`, not parallel-featured
  - use it to validate seam generality, not to start a second product branch

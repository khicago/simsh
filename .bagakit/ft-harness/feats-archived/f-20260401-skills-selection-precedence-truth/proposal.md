# Feat Proposal: f-20260401-skills-selection-precedence-truth

## Why
- The reference adapter already exposes `/skills` as a read-only projection with explicit `eligibility`, `precedence`, and an optional `selected` bit.
- That is enough to show metadata, but not enough to make selection trustworthy: today `selected` is still effectively adapter input, not a derived truth surface.
- The next adapter-side improvement should make skill selection explainable without widening core contracts or introducing a mutable skill registry.

## Goal
- Make /skills selection a derived, explainable adapter truth surface without widening core contracts or introducing a mutable skill registry.

## Scope
- In scope:
  - adapter-local selection derivation for `/skills` based on explicit eligibility and precedence inputs
  - explicit selection provenance and reason surfaces in `/skills/_index.json`, `/memory/projections.json`, and human-readable summaries
  - benchmark and adapter tests that prove winner/loser behavior deterministically
  - backlog and architecture updates that make the new selection contract canonical
- Out of scope:
  - writable `/skills` mounts or a product-style skill registry
  - remote sync, marketplace metadata, or semantic retrieval
  - moving skill selection semantics into core runtime packages
  - path-derived implicit competition scopes

## Impact
- Code paths:
  - `pkg/adapter/reference/adapter.go`
  - `pkg/adapter/reference/adapter_test.go`
  - `pkg/adapter/reference/adapter_helpers_test.go`
  - `benchmarks/simsh_native_reference/suite.go`
  - `docs/architecture-memory-skills-extension.md`
  - `docs/architecture-platform-adapter-contract.md`
  - `docs/notes-kernel-execution-backlog.md`
  - `.bagakit/long-run/bk-execution-handoff.md`
- Tests:
  - `go test ./pkg/adapter/reference ./benchmarks/simsh_native_reference ./pkg/engine/runtime`
  - `go test ./...`
- Rollout notes:
  - selection truth must stay adapter-local and read-only
  - the implementation should derive winner/loser state from explicit inputs and deterministic tie-breaks instead of hidden path heuristics

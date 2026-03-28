# Feat Proposal: f-20260403-adapter-mount-conformance-harness

## Why
- `K-022` extracted shared lifecycle/projection conformance, but mount-level invariants still live in scattered adapter-specific assertions.
- The seam still lacks one reusable proof layer for `VirtualMount` behavior itself: stable listing, recursive search, `DescribePath` metadata, and read-only access semantics.

## Goal
- Extract reusable VirtualMount conformance checks so multiple adapters prove stable list/search/describe/read-only metadata semantics without copying mount assertions into benchmark or adapter-local smoke tests.

## Scope
- In scope:
  - A shared `VirtualMount` conformance helper for deterministic list/search/describe/read-only metadata checks.
  - One focused conformance test in `reference` and one in `resourceset` that apply the helper to their projected mounts.
  - Backlog and contract docs that clarify mount conformance as a separate reusable seam layer from lifecycle conformance and benchmark validation.
- Out of scope:
  - Replacing `pkg/mount` unit tests.
  - Replacing benchmark scenarios.
  - Introducing write semantics into read-only mount conformance.

## Impact
- Code paths:
  - `pkg/adapter/internal/contracttest/`
  - `pkg/adapter/reference/*_test.go`
  - `pkg/adapter/resourceset/*_test.go`
- Tests:
  - shared adapter mount conformance tests
  - full repository gate
- Rollout notes:
  - Keep the helper mount-focused; adapter-specific workflow/skills/audit semantics remain adapter-local.

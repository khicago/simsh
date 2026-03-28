# Feat Proposal: f-20260403-adapter-seam-conformance-harness

## Why
- `reference` and `resourceset` now validate the adapter seam with materially different shapes, but their lifecycle and managed-memory assertions still live in adapter-specific tests and benchmark code.
- That duplicates seam rules and makes future adapters more likely to clone old smoke tests instead of proving the contract explicitly.

## Goal
- Extract reusable adapter conformance checks so richer and smaller adapters validate the same lifecycle, projection, and managed-memory seam without duplicating benchmark-specific logic.

## Scope
- In scope:
  - A shared adapter conformance test harness for lifecycle, projection mounts, opaque-state persistence, and managed-memory visibility.
  - One focused conformance test in `reference` and one in `resourceset` that use the harness while keeping their richer adapter-specific tests intact.
  - Contract and backlog docs that define why conformance belongs at the seam layer instead of inside the benchmark only.
- Out of scope:
  - Replacing the native reference benchmark.
  - Inventing new runtime contracts or adapter product semantics.
  - Adding a third adapter shape.

## Impact
- Code paths:
  - `pkg/adapter/<shared-test-package>/`
  - `pkg/adapter/reference/*_test.go`
  - `pkg/adapter/resourceset/*_test.go`
- Tests:
  - adapter lifecycle/seam tests
  - full repository gate
- Rollout notes:
  - Keep the harness narrow and invariant-driven; adapter-specific product behavior should stay in adapter-specific tests.

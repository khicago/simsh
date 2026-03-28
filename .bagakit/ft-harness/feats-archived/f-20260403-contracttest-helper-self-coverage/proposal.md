# Feat Proposal: f-20260403-contracttest-helper-self-coverage

## Why
- `K-022` and `K-023` extracted reusable adapter conformance helpers, but the new `pkg/adapter/internal/contracttest` package still has no direct package-local tests.
- Relying only on indirect adapter coverage leaves the reusable mechanism under-specified and makes failure semantics harder to trust.

## Goal
- Add direct tests for the shared adapter conformance helpers so the reusable contracttest layer itself has explicit success and failure coverage instead of relying only on indirect adapter execution.

## Scope
- In scope:
  - Direct tests for `contracttest` lifecycle and mount helpers.
  - Success and focused failure-path coverage for helper-level invariants.
  - Backlog and reusable-item docs that mark this layer as self-tested reusable mechanism rather than adapter-only incidental code.
- Out of scope:
  - New adapter features.
  - Benchmark expansion.
  - Reworking current adapter product semantics.

## Impact
- Code paths:
  - `pkg/adapter/internal/contracttest/*`
- Tests:
  - package-local helper tests
  - full repository gate
- Rollout notes:
  - Keep fake adapters/mounts minimal and purpose-built; do not build a second hidden runtime inside tests.

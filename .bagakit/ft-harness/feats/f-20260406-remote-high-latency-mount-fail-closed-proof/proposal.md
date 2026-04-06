# Feat Proposal: f-20260406-remote-high-latency-mount-fail-closed-proof

## Why
- `v0.3.0` now treats high-performance mount behavior as a release-gate issue, not a future nice-to-have.
- The fail-closed contract for `remote_high_latency` mounts already exists in docs and core dispatch code, but the direct proof layer is too thin to rely on it as release evidence.

## Goal
- Prove that remote_high_latency mounts explicitly refuse missing critical capabilities instead of silently degrading into fanout-heavy fallback behavior, and document the operator-facing contract.

## Scope
- In scope:
  - direct contract tests for missing-capability refusal on `remote_high_latency` mounts
  - engine or builtin-facing regressions that prove user-visible fail-closed behavior instead of optimistic fallback
  - docs and usage guidance for adapter authors and runtime callers
- Out of scope:
  - new cache layers
  - new mount feature families
  - widening remote mount behavior into a product-specific protocol

## Impact
- Code paths:
  - `pkg/contract/*`
  - `pkg/engine/*`
  - selected builtin tests under `pkg/builtin/*_test.go`
  - `docs/architecture-high-performance-mount-system.md`
  - `docs/notes-v0-2-x-to-v0-3-0-migration.md`
  - `docs/notes-kernel-execution-backlog.md`
- Tests:
  - `go test ./pkg/contract ./pkg/engine ./pkg/builtin ./pkg/mount -count=1`
  - `go test ./...`
- Rollout notes:
  - Keep the slice proof-oriented: strengthen evidence for the existing contract rather than broadening the mount API.

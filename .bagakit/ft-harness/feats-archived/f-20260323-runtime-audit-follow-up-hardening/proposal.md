# Feat Proposal: f-20260323-runtime-audit-follow-up-hardening

## Why
- Recent audit findings showed three remaining truth gaps:
  - multi-redirection write-limit behavior could still leave partial side effects,
  - projection metadata could diverge from canonical adapter names,
  - adapter benchmark success could pass business assertions without requiring full evidence.
- These are all “trust” problems, not feature gaps.

## Goal
- Eliminate remaining runtime truth gaps in redirection atomicity, adapter metadata canonicalization, and adapter benchmark evidence semantics.

## Scope
- In scope:
  - redirection atomicity under write-limited policy
  - canonical metadata lookup for adapter-side projections
  - benchmark success semantics that require evidence completeness
- Out of scope:
  - new shell features
  - new core contracts
  - broader adapter product expansion

## Impact
- Code paths:
  - `pkg/engine/script_runner.go`
  - `pkg/engine/engine_test.go`
  - `pkg/adapter/reference/adapter.go`
  - `pkg/adapter/reference/adapter_helpers_test.go`
  - `benchmarks/simsh_native_reference/`
- Tests:
  - `go test ./pkg/engine ./pkg/adapter/reference ./benchmarks/simsh_native_reference`
  - `go test ./...`
- Rollout notes:
  - keep fixes narrow and contract-driven; prefer invariant hardening over compatibility heuristics

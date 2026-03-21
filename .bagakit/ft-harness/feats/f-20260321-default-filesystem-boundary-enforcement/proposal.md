# Feat Proposal: f-20260321-default-filesystem-boundary-enforcement

## Why
- Default runtime filesystem implementations still have path-escape edge cases, which means capability claims are not yet strong enough to be treated as hard truth.
- Until the default boundary model is hardened, higher-level kernel work such as relative-path support or richer action surfaces would be building on top of an untrusted base.

## Goal
- Harden default runtime filesystem boundary enforcement so capability claims and actual write/remove behavior cannot diverge through path escape edge cases.

## Scope
- In scope:
- hardening default path enforcement in default filesystem implementations
- fixing known symlink and nested-descendant escape shapes
- verifying mount/synthetic path mutation attempts fail consistently where expected
- adding regression coverage for boundary edge cases
- Out of scope:
- broader action-surface expansion
- virtual `cwd` or relative-path support
- trace schema redesign beyond what is needed to validate boundary behavior

## Impact
- Code paths:
- `pkg/adapter/agentfs/filespace_core.go`
- `pkg/adapter/localfs/adapter.go`
- related filesystem and engine tests
- Tests:
- regression tests for escape writes/removes
- mount/synthetic path mutation rejection tests
- Rollout notes:
- this feat is the prerequisite for path-resolution ergonomics and should finish before relative-path work begins

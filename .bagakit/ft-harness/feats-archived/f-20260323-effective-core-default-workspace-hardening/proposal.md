# Feat Proposal: f-20260323-effective-core-default-workspace-hardening

## Why
- Strict kernel contracts are now relatively mature, but agent-visible quality still depends on the default workspace surface.
- The highest remaining leverage is in `pkg/engine` and `pkg/builtin`, where pipe behavior, structured modes, confirmation output, and list/query surfaces can regress without obvious compiler pressure.
- This feat exists to harden those default-workspace seams before spending effort on entry-surface ergonomics.

## Goal
- Raise default-workspace confidence in engine+builtin behavior without expanding product surface.

## Scope
- In scope:
  - regression coverage for high-value default-workspace behaviors
  - engine-level checks for builtin output contracts and composition behavior
  - tests for explicit structured modes and low-noise confirmation modes
- Out of scope:
  - adding broad new command surface area
  - CLI/HTTP product ergonomics
  - adapter-specific product logic

## Impact
- Code paths:
  - `pkg/engine/engine_test.go`
  - selected builtin-facing docs or backlog notes if the contract needs clarification
- Tests:
  - `go test ./pkg/engine ./pkg/builtin`
- Rollout notes:
  - prefer additive regression guards over semantic rewrites

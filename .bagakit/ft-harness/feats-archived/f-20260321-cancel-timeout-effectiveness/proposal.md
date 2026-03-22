# Feat Proposal: f-20260321-cancel-timeout-effectiveness

## Why
- Policy timeout already exists in contracts, but many filesystem and execution paths still ignore `ctx`, which makes interruption semantics weaker than they appear.
- Long-running agent work needs timeout and cancel controls that have real operational effect, not only configuration presence.

## Goal
- Audit and improve cancellation and timeout effectiveness across execution and filesystem paths so policy interruption controls have real operational effect.

## Scope
- In scope:
- auditing context use across execution and default filesystem paths
- improving cancellation effectiveness where long-running work should stop cooperatively
- ensuring timeout/cancel signals are visible enough for runtime consumers
- Out of scope:
- full multi-agent orchestration
- session redesign unrelated to interruption behavior
- unrelated ACI expansions

## Impact
- Code paths:
- `pkg/engine/execution_result.go`
- `pkg/adapter/agentfs/filespace_core.go`
- `pkg/adapter/localfs/adapter.go`
- related tests around cancellation and timeout behavior
- Tests:
- timeout/cancel execution tests
- context-aware filesystem operation tests where practical
- Rollout notes:
- prefer incremental truth: land effective interruption where it matters most first, then widen coverage

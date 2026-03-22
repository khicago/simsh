# Feat Proposal: f-20260301-session-lifecycle-policy-ceiling

## Why
- `simsh` still treats execution as mostly one-shot, so there is no first-class session boundary for resume, checkpoint, or cross-call runtime state.
- Policy is currently attached to each runtime construction call, which makes narrowing and explicit elevation rules hard to model across a long-running agent interaction.

## Goal
- Add first-class session lifecycle primitives and session-scoped policy ceiling semantics for cross-call runtime state.

## Scope
- In scope:
- contract-level session identifiers and lifecycle hooks (`create`, `resume`, `checkpoint`, `close`)
- session-scoped policy ceiling and per-execution narrowing semantics
- preserving one-shot execution as an ephemeral session path
- Out of scope:
- structured execution result payload redesign
- execution trace side-effect collection
- adapter-specific memory or projection behavior

## Impact
- Code paths:
- `pkg/contract`
- `pkg/engine/runtime`
- `pkg/service/httpapi`
- `cmd/simsh-cli`
- Tests:
- session lifecycle contract tests
- policy narrowing/elevation behavior tests
- Rollout notes:
- keep legacy one-shot calls working while introducing optional session-aware APIs

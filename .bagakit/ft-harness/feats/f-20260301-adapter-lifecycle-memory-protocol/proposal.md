# Feat Proposal: f-20260301-adapter-lifecycle-memory-protocol

## Why
- Kernel contracts alone do not make adapters consistent; lifecycle hooks and optional memory semantics need one shared platform contract.
- If adapters each define their own hydrate/checkpoint/observe rules, session and trace work will still fragment at the seam.

## Goal
- Add adapter lifecycle hooks, optional memory hydrate-observe-checkpoint-close protocol, and explicit adapter-side session integration points.

## Scope
- In scope:
- adapter hooks for session create/resume/checkpoint/close
- optional memory lifecycle protocol for adapter-projected `/memory` views
- explicit trace-consumption and projection responsibilities at the adapter boundary
- Out of scope:
- production business adapter implementation
- new core domain semantics beyond generic adapter hooks
- multi-agent orchestration

## Impact
- Code paths:
- `pkg/contract`
- `pkg/engine/runtime`
- `pkg/fs`
- adapter-facing docs and interfaces
- Tests:
- adapter lifecycle hook tests
- optional memory protocol contract tests
- failure-mode tests around projection freshness and partial materialization
- Rollout notes:
- treat `/memory` as adapter-managed projection, not as an implicitly writable core zone

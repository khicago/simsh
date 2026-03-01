---
title: Core vs adapter contract split for next-stage runtime design
kind: decision
confidence: medium
tags:
  - architecture
  - session
  - trace
  - adapters
sources:
  - docs/notes-project-charter.md
  - docs/architecture-session-trace-model.md
  - docs/architecture-platform-adapter-contract.md
created: 2026-03-01T00:00:00Z
updated: 2026-03-01T00:00:00Z
---

## Context
The project needed a cleaner separation between generic runtime work and platform-adapter responsibilities while planning session, trace, and memory-related evolution.

## Decision
- Keep `simsh` core generic and lightweight.
- Promote `Session`, `ExecutionResult`, and `ExecutionTrace` to core contract work.
- Keep RPC-to-file projection and memory lifecycle semantics at the platform-adapter boundary.
- Validate new contracts with at least one real adapter-backed workload before treating them as stable.

## Rationale
- Core needs machine-consumable execution contracts regardless of business domain.
- Adapter seams are where integration drift and workload mismatch are most likely.
- Mixing both concerns in one doc or one layer makes future changes harder to reason about.

## Scope
- Applies to next-stage runtime design work after the current v1 baseline.
- Does not force any single business domain into the core runtime.

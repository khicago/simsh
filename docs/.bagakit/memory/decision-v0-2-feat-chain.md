---
title: Complete v0.2 contract feat chain with reference adapter workload
kind: decision
confidence: high
tags:
  - decision
  - contracts
  - adapters
  - v0-2
sources:
  - .bagakit/ft-harness/index/FEATS_DAG.json
  - pkg/contract/session_contract.go
  - pkg/contract/execution_result.go
  - pkg/contract/adapter_contract.go
  - pkg/adapter/reference/adapter.go
  - pkg/adapter/reference/adapter_test.go
created: 2026-03-01T19:01:10Z
updated: 2026-03-23T00:00:00Z
---

## Context
The repo needed a stable way to describe how the v0.2 runtime contract actually matured, without forcing later work to rediscover the dependency order from scattered feat history.

## Decision
The v0.2 contract chain was implemented as five linked feat units on `main`:

- `f-20260301-session-lifecycle-policy-ceiling`
- `f-20260301-structured-execution-result-contract`
- `f-20260301-execution-trace-side-effect-tracking`
- `f-20260301-adapter-lifecycle-memory-protocol`
- `f-20260301-reference-adapter-e2e-validation`

## Stable Layering
- session lifecycle
- structured execution result
- execution trace
- adapter lifecycle and memory protocol
- reference adapter workload

## Rationale
- Session lifecycle had to land first so cross-call runtime state and policy ceilings were explicit.
- Structured result and trace then turned execution into a machine-consumable contract instead of text-only behavior.
- Adapter lifecycle and memory protocol came after that because adapters need stable session and trace hooks to project managed state safely.
- The reference adapter was the proof step: it showed that opaque session state plus execution traces are enough to rebuild adapter projections and feed memory observations across create, checkpoint, close, and resume.

## Scope
- Use this as the durable ordering rule when explaining why v0.2 contracts look the way they do.
- Future adapter work should extend from this seam rather than pulling product-specific lifecycle semantics back into core runtime packages.

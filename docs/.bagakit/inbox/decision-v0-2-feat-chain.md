---
title: Complete v0.2 contract feat chain with reference adapter workload
kind: decision
status: inbox
tags:
  - decision
sources:
  - .bagakit/ft-harness/index/FEATS_DAG.json
  - pkg/contract/session_contract.go
  - pkg/contract/execution_result.go
  - pkg/contract/adapter_contract.go
  - pkg/adapter/reference/adapter.go
  - pkg/adapter/reference/adapter_test.go
created: 2026-03-01T19:01:10Z
---

## Candidate
The planned v0.2 contract chain is now implemented in code on `main` as five independent feat commits:

- `f-20260301-session-lifecycle-policy-ceiling`
- `f-20260301-structured-execution-result-contract`
- `f-20260301-execution-trace-side-effect-tracking`
- `f-20260301-adapter-lifecycle-memory-protocol`
- `f-20260301-reference-adapter-e2e-validation`

Durable takeaway:

- The stable layering is now session lifecycle -> structured execution result -> execution trace -> adapter lifecycle/memory protocol -> reference adapter workload.
- The reference adapter is the first concrete proof that adapter projections can rebuild from opaque session state and consume execution traces into memory observations across create, checkpoint, close, and resume.
- Future adapter work should extend from this seam instead of introducing product-specific lifecycle semantics directly into core runtime packages.

## Promote To
- `docs/.bagakit/memory/decision-v0-2-feat-chain.md` (curated), or
- `docs/<type>-<topic>.md` (normative/deep guide)

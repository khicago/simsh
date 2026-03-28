---
title: Reference projection materialization should stay separate from freshness
kind: howto
confidence: high
tags:
  - adapter
  - projection
  - materialization
  - freshness
sources:
  - pkg/adapter/reference/adapter.go
  - pkg/adapter/reference/adapter_test.go
  - benchmarks/simsh_native_reference/suite.go
  - docs/architecture-platform-adapter-contract.md
created: 2026-04-01T00:00:00Z
updated: 2026-04-01T00:00:00Z
---

## Context

The reference adapter needed one more layer of truth after freshness and curation: a caller should be able to distinguish “stale but present” from “partial” from “failed” without overloading freshness labels.

## Guidance

- Keep freshness and materialization separate.
- Use freshness for update state such as `snapshot`, `live`, `stale`, `updated`.
- Use materialization for presence/completeness such as `materialized`, `partial`, `failed`.
- Keep failed projections visible in machine-readable index/projection views even if their mounted file bodies are absent.
- Make partial/error detail machine-readable via `reason` or an equivalent failure detail field.
- Benchmark this through structured projection decoding, not message-only checks.

## Scope

- Applies to adapter-side projection records and future projection failure modeling.
- Does not require new core-runtime contracts.

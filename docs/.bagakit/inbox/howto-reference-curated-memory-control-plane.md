---
title: Reference adapter curated memory should come from explicit control-plane actions, not trace-derived implicit summaries
kind: howto
confidence: high
tags:
  - adapter
  - memory
  - curation
  - control-plane
sources:
  - pkg/adapter/reference/adapter.go
  - pkg/adapter/reference/adapter_test.go
  - benchmarks/simsh_native_reference/suite.go
  - docs/architecture-platform-adapter-contract.md
  - docs/architecture-memory-skills-extension.md
created: 2026-04-01T00:00:00Z
updated: 2026-04-01T00:00:00Z
---

## Context

The first explicit curated-memory slice for the reference adapter needed to stay aligned with the adapter contract: `/memory` is a read-only managed view, while curation is an explicit control-plane responsibility.

## Guidance

- Do not auto-generate curated entries from raw trace sets; that collapses curation back into another derived report.
- Keep curated entries explicit, stable, and machine-readable, with at least:
  - stable `id`
  - human-readable `title`
  - `source_paths`
  - adapter-local provenance such as `source` and `revision`
- Persist curated entries through adapter opaque state so checkpoint/resume keeps them intact.
- Keep curated views distinct from raw observations and projection indexes.
- Expose curated views through stable read-only files such as `/memory/curated.json` and `/memory/curated.md`.

## Scope

- Applies to the reference adapter and future managed-memory work.
- Does not imply a product memory engine, retrieval system, or writable `/memory` surface.

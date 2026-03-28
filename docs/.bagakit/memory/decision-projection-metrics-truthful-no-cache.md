---
title: Projection metrics should stay truthful and avoid fake cache semantics
kind: decision
tags:
  - decision
  - adapter
  - observability
  - metrics
sources:
  - pkg/adapter/reference/adapter.go
  - pkg/adapter/reference/adapter_helpers_test.go
  - pkg/adapter/reference/adapter_test.go
  - benchmarks/simsh_native_reference/suite.go
  - docs/architecture-memory-skills-extension.md
  - docs/architecture-platform-adapter-contract.md
created: 2026-04-02T18:16:43Z
confidence: low
updated: 2026-04-02T18:19:14Z
---

## Candidate
Context:
- During `K-020`, the reference adapter needed projection metrics and denial surfaces, but it still had no actual caching layer.
- The tempting shortcut was to invent cache-hit fields just to satisfy Stage D language from the architecture doc.

Decision:
- Projection metrics should report only truth the adapter actually has: generation, build latency, projection counts, freshness, materialization, control-plane event counts, and unique denied-path counts.
- Cache-oriented fields stay explicit but negative, for example `cache_hit_metrics_available: false`, until a real cache exists.
- Denial surfaces should classify only what the adapter can prove by namespace or managed prefix; anything else stays `external_or_unknown`.

Why:
- This keeps observability trustworthy instead of decorative.
- It prevents fake telemetry from becoming accidental contract.
- It preserves a clear boundary between current reference behavior and future optimization work such as real caching.

## Promote To
- `docs/.bagakit/memory/decision-projection-metrics-truthful-no-cache.md` (curated), or
- `docs/<type>-<topic>.md` (normative/deep guide)

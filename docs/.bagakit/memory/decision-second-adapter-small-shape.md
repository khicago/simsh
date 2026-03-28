---
title: Second adapter validation should stay smaller than reference
kind: decision
tags:
  - decision
  - adapter
  - seam
sources:
  - pkg/adapter/resourceset/adapter.go
  - pkg/adapter/resourceset/adapter_test.go
  - benchmarks/simsh_native_reference/suite.go
  - docs/architecture-platform-adapter-contract.md
  - docs/notes-kernel-execution-backlog.md
created: 2026-04-03T07:05:10Z
confidence: low
updated: 2026-04-03T07:08:15Z
---

## Candidate
Context:
- During `K-021`, the easiest way to “validate a second adapter” would have been to copy most of the `reference` adapter and remove a few features.
- That would have produced another adapter, but it would not have proven that the seam works for a materially smaller shape.

Decision:
- The second adapter used for seam validation should stay smaller than `reference`.
- It should prove lifecycle + projection + minimal managed memory, not feature parity.
- If it starts accreting skills, workflows, curation, rich control-plane semantics, or adapter-local audit/metrics clones, it stops being seam validation and starts becoming a second product branch.

Why:
- A smaller adapter is a stronger proof that the seam is generic.
- It makes hidden reference-specific assumptions visible earlier.
- It keeps the cost of the validation slice low and the conclusion clearer.

## Promote To
- `docs/.bagakit/memory/decision-second-adapter-small-shape.md` (curated), or
- `docs/<type>-<topic>.md` (normative/deep guide)

---
title: Composition stress should reuse existing truth surfaces
kind: decision
tags:
  - decision
  - adapter
  - benchmark
  - seam
sources:
  - pkg/adapter/reference/adapter_test.go
  - benchmarks/simsh_native_reference/suite.go
  - docs/architecture-platform-adapter-contract.md
  - docs/notes-kernel-execution-backlog.md
created: 2026-04-03T12:00:00Z
confidence: medium
updated: 2026-04-03T12:00:00Z
---

## Candidate
Context:
- After `K-022` to `K-024`, seam, mount, and helper layers were individually strong.
- The next risk was composition drift across those already-existing truth surfaces, not lack of new capabilities.

Decision:
- Composition/evolution stress validation should reuse existing machine-readable truth surfaces.
- Prefer `/memory/status.json`, `/memory/projections.json`, `/memory/projection_metrics.json`, `/memory/skills_audit.json`, `/memory/denials.json`, and `/memory/workflows.json`.
- Do not add new adapter nouns or product APIs just to make the stress scenario broader.

Why:
- It keeps the scenario proof-oriented rather than feature-oriented.
- It pressures the actual SSOT surfaces that planners and harnesses already consume.
- It avoids quietly turning a validation slice into a new product branch.

## Promote To
- `docs/.bagakit/memory/decision-composition-stress-reuses-existing-truth-surfaces.md` (curated), or
- `docs/<type>-<topic>.md` (normative/deep guide)

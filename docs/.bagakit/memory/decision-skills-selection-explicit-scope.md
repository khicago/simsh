---
title: Skill selection should use explicit scopes, not path-derived competition
kind: decision
tags:
  - decision
  - adapter
  - skills
  - selection
sources:
  - pkg/adapter/reference/adapter.go
  - pkg/adapter/reference/adapter_helpers_test.go
  - pkg/adapter/reference/adapter_test.go
  - benchmarks/simsh_native_reference/suite.go
  - docs/architecture-memory-skills-extension.md
  - docs/architecture-platform-adapter-contract.md
  - docs/notes-kernel-execution-backlog.md
created: 2026-04-01T18:47:06Z
updated: 2026-04-01T18:58:00Z
confidence: low
---

## Candidate
Context:
- During the `K-017` `/skills` selection-truth implementation, the tempting shortcut was to infer competition groups from path layout such as `planning/*`.
- That would have made the reference adapter look smart quickly, but it would have baked product semantics into file layout and made selection truth harder to audit.

Decision:
- Skill competition must be declared explicitly by adapter metadata such as `SelectionScope`.
- Path hierarchy is not a fallback selection scope.
- Unscoped skills may still surface explicit or singleton selection state, but they must not silently compete because they happen to share a directory prefix.
- Keep the compatibility `selected` bit, but pair it with explicit `selection` provenance so losers and ineligible skills remain visible and explainable.

Why:
- This keeps selection truth adapter-local and auditable.
- It avoids turning projection path layout into hidden policy.
- It keeps `/skills` read-only and compatible with multiple harness-specific grouping models.
- It preserves a clean boundary between generic kernel path semantics and adapter-specific skill policy.
- It lets adapter tests and the reference benchmark consume one shared selection truth surface instead of embedding another ranking heuristic in each caller.

## Promote To
- `docs/.bagakit/memory/decision-skills-selection-explicit-scope.md` (curated), or
- `docs/<type>-<topic>.md` (normative/deep guide)

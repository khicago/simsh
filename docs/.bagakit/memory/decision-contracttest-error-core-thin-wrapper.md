---
title: Contracttest helpers should keep error-returning cores
kind: decision
tags:
  - decision
  - testing
  - coverage
  - seam
sources:
  - pkg/adapter/internal/contracttest/lifecycle.go
  - pkg/adapter/internal/contracttest/mount.go
  - pkg/adapter/internal/contracttest/lifecycle_test.go
  - pkg/adapter/internal/contracttest/mount_test.go
  - docs/architecture-platform-adapter-contract.md
  - docs/notes-kernel-execution-backlog.md
created: 2026-04-03T11:30:00Z
confidence: medium
updated: 2026-04-03T11:30:00Z
---

## Candidate
Context:
- `K-022` and `K-023` introduced reusable test helpers under `pkg/adapter/internal/contracttest`.
- Once `K-024` tried to add direct coverage, the hard part was not success-path testing but exercising helper failure semantics without subprocess tricks or fake mega-adapters.

Decision:
- Keep reusable `contracttest` helpers as `error`-returning cores with thin `testing.T` wrappers at the edge.
- Direct tests should target the `error`-returning layer.
- Adapter-local tests may still use the thin wrappers for readability.

Why:
- It makes failure semantics directly testable.
- It keeps package-local coverage meaningful instead of forcing subprocess or branch-farming tricks.
- It preserves a small reusable core while keeping call sites ergonomic.

## Promote To
- `docs/.bagakit/memory/decision-contracttest-error-core-thin-wrapper.md` (curated), or
- `docs/<type>-<topic>.md` (normative/deep guide)

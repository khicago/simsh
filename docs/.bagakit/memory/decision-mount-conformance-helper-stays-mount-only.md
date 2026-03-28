---
title: Shared mount conformance helper should stay mount-only
kind: decision
tags:
  - decision
  - adapter
  - testing
  - mount
sources:
  - pkg/adapter/internal/contracttest/mount.go
  - pkg/adapter/reference/adapter_conformance_test.go
  - pkg/adapter/resourceset/adapter_test.go
  - docs/architecture-platform-adapter-contract.md
  - docs/notes-kernel-execution-backlog.md
created: 2026-04-03T10:10:00Z
confidence: medium
updated: 2026-04-03T10:10:00Z
---

## Candidate
Context:
- After `K-022`, lifecycle conformance was reusable, but mount-level assertions were still duplicated across adapters.
- The tempting shortcut for `K-023` was to build a broader filesystem-style helper that also absorbed benchmark or product assertions.

Decision:
- Keep the shared mount conformance helper mount-only.
- It should own only `VirtualMount` invariants: `Exists`, deterministic list/search behavior, `DescribePath`, and read-only capability truth.
- Product-shaped assertions such as workflows, skill selection, audit, metrics, or runtime write-denial behavior must stay in adapter-local tests or benchmark layers.

Why:
- A narrow helper strengthens the seam without creating a second filesystem DSL inside tests.
- It keeps lifecycle conformance, mount conformance, and benchmark validation as separate proof layers.
- It makes adapter-local failures easier to classify because mount failures no longer hide inside lifecycle or benchmark smoke tests.

## Promote To
- `docs/.bagakit/memory/decision-mount-conformance-helper-stays-mount-only.md` (curated), or
- `docs/<type>-<topic>.md` (normative/deep guide)

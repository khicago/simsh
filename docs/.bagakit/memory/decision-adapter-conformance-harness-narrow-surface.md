---
title: Shared adapter conformance harness should stay narrow
kind: decision
tags:
  - decision
  - adapter
  - testing
  - seam
sources:
  - pkg/adapter/internal/contracttest/lifecycle.go
  - pkg/adapter/reference/adapter_conformance_test.go
  - pkg/adapter/resourceset/adapter_test.go
  - docs/architecture-platform-adapter-contract.md
  - docs/notes-kernel-execution-backlog.md
created: 2026-04-03T07:43:21Z
confidence: medium
updated: 2026-04-03T07:43:21Z
---

## Candidate
Context:
- After `reference` and `resourceset` both proved the adapter seam, the remaining risk was not missing coverage but duplicated seam logic drifting across adapter-specific tests.
- The easy mistake at this point would have been to build a generic testing DSL that slowly absorbs benchmark and product assertions.

Decision:
- Keep the shared adapter conformance harness narrow.
- It should own only the reusable seam invariants: lifecycle sequencing, mount presence, opaque-state persistence, and managed-memory visibility helpers.
- Product-shaped assertions such as workflow behavior, skill selection, audit, metrics, or benchmark scenario semantics must stay in adapter-local tests or the benchmark layer.

Why:
- A narrow harness strengthens the seam without inventing a second product model.
- It lowers copy-paste pressure for future adapters while keeping the abstraction obvious.
- Explicit adapter-local callbacks are easier to reason about than reflection-heavy generic test DSLs.

## Promote To
- `docs/.bagakit/memory/decision-adapter-conformance-harness-narrow-surface.md` (curated), or
- `docs/<type>-<topic>.md` (normative/deep guide)

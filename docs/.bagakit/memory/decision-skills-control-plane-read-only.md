---
title: Skill control-plane updates should refresh projections, not make /skills writable
kind: decision
tags:
  - decision
  - adapter
  - skills
  - control-plane
sources:
  - pkg/adapter/reference/adapter.go
  - pkg/adapter/reference/adapter_test.go
  - benchmarks/simsh_native_reference/suite.go
  - docs/architecture-memory-skills-extension.md
  - docs/architecture-platform-adapter-contract.md
created: 2026-04-01T19:20:02Z
updated: 2026-04-01T19:23:00Z
confidence: low
---

## Candidate
Context:
- During the `K-018` skill control-plane lifecycle slice, the easiest implementation path would have been to make `/skills` writable or to hot-swap the visible mount immediately after adapter API calls.
- That would blur projection and control-plane responsibilities and create a second mutation path outside the adapter lifecycle.

Decision:
- Skill evolution stays behind explicit adapter-local control-plane APIs such as add, update, and remove.
- `/skills` remains read-only from session command execution.
- Control-plane updates become visible on the next projection rebuild or resume, not by mutating the live mount in place.

Why:
- This preserves a clean separation between projection surface and control plane.
- It keeps session-visible behavior auditable and lifecycle-aware.
- It lets skill reselection continue to flow through one adapter-local SSOT path instead of mixing runtime mutation with policy logic.
- It avoids turning a reference adapter seam into a hidden registry or writable backdoor.

## Promote To
- `docs/.bagakit/memory/decision-skills-control-plane-read-only.md` (curated), or
- `docs/<type>-<topic>.md` (normative/deep guide)

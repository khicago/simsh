---
title: Skill control-plane audit should be machine-readable and visibility-explicit
kind: decision
tags:
  - decision
  - adapter
  - skills
  - audit
sources:
  - pkg/adapter/reference/adapter.go
  - pkg/adapter/reference/adapter_helpers_test.go
  - pkg/adapter/reference/adapter_test.go
  - benchmarks/simsh_native_reference/suite.go
  - docs/architecture-memory-skills-extension.md
  - docs/architecture-platform-adapter-contract.md
created: 2026-04-02T04:15:17Z
confidence: low
updated: 2026-04-02T04:16:48Z
---

## Candidate
Context:
- During `K-019`, the reference adapter already had a real skill control plane, but nothing machine-readable described what changed or when those changes became projection-visible.
- The tempting shortcut was to rely on projection diffs or prose logs. That would have created a second, weaker source of truth for control-plane behavior.

Decision:
- Skill control-plane audit should be exposed as a compact machine-readable view plus a compact human-readable summary.
- Visibility timing must be explicit; callers should not have to guess from lifecycle side effects or mount diffs.
- Audit remains adapter-local and should describe control-plane mutations, not denied shell writes to `/skills`.

Why:
- This keeps control-plane truth and projection truth aligned without building a generic event bus.
- It preserves the read-only `/skills` boundary while still making add/update/remove behavior observable.
- It gives benchmarks and harnesses a stable surface for verifying winner/loser changes and visibility timing.
- It avoids turning ad hoc logging into a hidden contract.

## Promote To
- `docs/.bagakit/memory/decision-skills-audit-machine-readable.md` (curated), or
- `docs/<type>-<topic>.md` (normative/deep guide)

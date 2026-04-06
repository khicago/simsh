---
title: Paired uplift baseline stays repo-controlled and thin, not host-shell ambient
kind: decision
status: inbox
tags:
  - decision
  - benchmark
  - uplift
  - runtime
sources:
  - docs/architecture-paired-ab-uplift-proof-harness.md
  - benchmarks/paired_uplift/task_set.json
  - benchmarks/paired_uplift/substrate.go
  - benchmarks/paired_uplift/reports/raw-baseline-20260406.json
updated: 2026-04-06T05:55:00Z
created: 2026-04-06T05:55:00Z
---

## Candidate
Context:
- 2026-04-06 K-030 implementation needed a baseline substrate for paired A/B uplift proof.
- The tempting shortcut was to compare full `simsh` against the ambient host shell.

Decision:
- Keep the paired uplift baseline repo-controlled.
- The first baseline is `thin_core_stateless`, not a workstation shell.
- The baseline is intentionally thinner by contract:
  - no `json`
  - no `rg`
  - no session-scoped cwd continuity

Rationale:
- Ambient host-shell command availability would contaminate the comparison with machine-local PATH drift.
- K-030 is supposed to vary only runtime substrate, not workstation setup luck.
- A repo-controlled thin baseline keeps the A/B claim explainable and reproducible.

Scope:
- Applies to K-030 paired uplift work and future baseline-expansion decisions.
- Does not forbid future additional baselines, but each new baseline should be explicit and repo-controlled.

## Promote To
- `docs/.bagakit/memory/decision-paired-uplift-baseline-repo-controlled.md` (curated), or
- `docs/<type>-<topic>.md` (normative/deep guide)

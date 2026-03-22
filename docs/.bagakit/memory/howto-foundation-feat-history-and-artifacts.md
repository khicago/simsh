---
title: How to read the 2026 foundation feat history and find evidence
kind: howto
confidence: high
tags:
  - howto
  - feats
  - kernel
  - recall
sources:
  - docs/notes-kernel-optimization-plan.md
  - docs/notes-kernel-execution-backlog.md
  - docs/notes-builtin-aci-review.md
  - .bagakit/ft-harness/feats-archived/f-20260217-vfs-mount-ancestors/summary.md
  - .bagakit/ft-harness/feats-archived/f-20260218-path-access-metadata/summary.md
  - .bagakit/ft-harness/feats-archived/f-20260222-memory-skills-extension-framework/summary.md
  - .bagakit/ft-harness/feats-archived/f-20260321-default-filesystem-boundary-enforcement/summary.md
  - .bagakit/ft-harness/feats-archived/f-20260321-virtual-cwd-path-resolution/summary.md
  - .bagakit/ft-harness/feats-archived/f-20260321-mutation-trace-fidelity/summary.md
  - .bagakit/ft-harness/feats-archived/f-20260321-cancel-timeout-effectiveness/summary.md
  - .bagakit/ft-harness/feats-archived/f-20260322-simsh-native-reference-validation-metric-gates/summary.md
  - .bagakit/ft-harness/feats-archived/f-20260322-builtin-aci-dual-readable-query-tooling/summary.md
created: 2026-03-23T00:00:00Z
updated: 2026-03-23T00:00:00Z
---

## Context
The repo shipped a dense wave of feat work across 2026-02 and 2026-03. The archived feat summaries are the raw evidence, but recall quality gets worse if each feat also leaves behind a separate memory placeholder.

## Foundation Waves

### Early foundation wave
- `f-20260217-vfs-mount-ancestors`: made mount parents reachable while keeping mount and synthetic paths mutation-safe.
- `f-20260218-path-access-metadata`: established path access/capability metadata as a surfaced SSOT in listings and APIs.
- `f-20260220-tooling-docs-consistency`: tightened the contract between tool behavior, manuals, and README.
- `f-20260222-memory-skills-extension-framework`: kept memory and skills outside core, projecting them through adapters and mounts.
- `f-20260224-plan-sync-agentfs-write-limit`: made policy-backed write limits real in default filesystem behavior.

### Kernel hardening wave
- `f-20260321-default-filesystem-boundary-enforcement`
- `f-20260321-virtual-cwd-path-resolution`
- `f-20260321-mutation-trace-fidelity`
- `f-20260321-cancel-timeout-effectiveness`

This wave made the runtime much more trustworthy for agent work: boundary truth, relative navigation, trace truth, and operational interruption.

### Proof and interface wave
- `f-20260322-simsh-native-reference-validation-metric-gates`
- `f-20260322-builtin-aci-dual-readable-query-tooling`

This wave answered the next two questions:
- does the kernel measurably help realistic agent file workflows?
- does the default builtin ACI reduce parse cost and ambiguity enough to matter?

## Where To Look
1. Start with the canonical docs if you need current design intent:
   - `docs/notes-kernel-optimization-plan.md`
   - `docs/notes-kernel-execution-backlog.md`
   - `docs/notes-builtin-aci-review.md`
   - relevant `docs/architecture-*.md`
2. Open the archived feat summary if you need delivery evidence:
   - `.bagakit/ft-harness/feats-archived/<feat>/summary.md`
   - then `proposal.md` and `tasks.md` if you need rollout framing
3. Open the linked code and tests once you know which feat and which canonical doc own the concept.

## Practical Rule
- Use curated memory for the storyline and lookup pattern.
- Use canonical docs for the current design.
- Use archived feat artifacts for validation and delivery evidence.
- Do not recreate one memory file per feat unless it adds knowledge that the canonical docs and archived feat summaries do not already preserve.

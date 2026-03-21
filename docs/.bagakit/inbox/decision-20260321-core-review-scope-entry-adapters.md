---
title: Review scope: keep CLI and HTTP as later entry-adapter follow-ups
kind: decision
status: inbox
tags:
  - decision
sources:
  - docs/architecture.md
  - docs/notes-project-charter.md
  - docs/notes-v0-1-0-to-v0-2-migration.md
  - README.md
created: 2026-03-21T15:37:05Z
---

## Candidate
Context:
- 2026-03-21 review scope was reset after clarifying that `simsh` is fundamentally a lightweight virtual shell/runtime for agents, not a CLI-first or HTTP-first product.

Decision:
- Treat `pkg/contract`, `pkg/sh`, `pkg/fs`, and `pkg/engine/runtime` as the kernel review scope.
- Treat CLI/TUI and HTTP as entry adapters around the kernel, not the source of truth for architecture or trust-boundary decisions.

Why:
- The charter defines `simsh` as a reusable deterministic runtime core for higher-level platforms.
- The architecture and migration docs already frame CLI/TUI and HTTP as consumer surfaces with derivative pressure relative to embedders and adapters.
- Reviewing entry adapters first risks overfitting kernel design to convenience APIs instead of stable runtime contracts.

Later optimization direction for entry adapters:
- Keep them thin over the unified runtime stack.
- Align them with structured result/trace and session/policy contracts from the kernel.
- Improve boundary and integration tests there after the kernel model is reviewed and stabilized.

## Promote To
- `docs/.bagakit/memory/decision-20260321-core-review-scope-entry-adapters.md` (curated), or
- `docs/<type>-<topic>.md` (normative/deep guide)

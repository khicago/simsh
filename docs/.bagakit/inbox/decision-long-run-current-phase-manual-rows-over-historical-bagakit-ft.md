---
title: Current-phase long-run should use manual rows when only historical bagakit-ft items remain
kind: decision
confidence: high
tags:
  - long-run
  - bagakit-ft
  - planning
  - historical
sources:
  - .bagakit/long-run/bk-execution-table.json
  - .bagakit/long-run/feature-list.json
  - .bagakit/long-run/next-action.json
  - docs/notes-kernel-execution-backlog.md
created: 2026-03-31T08:45:00Z
updated: 2026-03-31T08:45:00Z
---

## Context

By 2026-03-31, the archived March `bagakit-ft` rows still showed up as blocked historical evidence in long-run generated state even though the active repository work had moved on to adapter-side realism planning.

## Decision

- Treat the old March `bagakit-ft` rows as historical evidence only, not as the active current-phase backlog.
- Rebuild the current phase with explicit `manual` rows when there is no active feat/task chain to execute.
- Make the new manual rows alternate `IMPLEMENT -> REVIEW -> REFINE -> IMPLEMENT` so long-run does not degenerate into repeated coding-only passes.

## Rationale

- The archived March feat chain no longer represents the next highest-value work, so leaving it active distorts `next-action` selection.
- Current-phase work is centered on adapter-side non-core realism rather than unfinished kernel foundation items.
- Manual rows are the lightest valid long-run contract for planning slices that cut across implementation, review, and design refinement.

## Scope

- Applies when the repo has no active feat/task chain but still has historical `bagakit-ft` evidence that should remain searchable.
- Does not forbid re-enabling `bagakit-ft` later if a new current-phase feat chain is created.
